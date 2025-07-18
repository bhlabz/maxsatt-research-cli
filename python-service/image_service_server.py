import grpc
from concurrent import futures
import image_service_pb2
import image_service_pb2_grpc
import json
from request_image import search_scenes, BAND_MAP, crop_band, download_band, S3_BUCKET
import tempfile
import rasterio
import boto3
import os
import logging
from shapely.geometry import shape

logging.basicConfig(level=logging.INFO)

class ImageServiceServicer(image_service_pb2_grpc.ImageServiceServicer):
    def ListAvailableDates(self, request, context):
        logging.info(f"ListAvailableDates called with start_date={request.start_date}, end_date={request.end_date}, bands={request.bands}, geojson_features={request.geojson_features}")
        # Parse all GeoJSON features
        try:
            geometries = [json.loads(g) for g in request.geojson_features]
            shapes = [shape(geom) for geom in geometries]
            from shapely.ops import unary_union
            union_geom = unary_union(shapes)
            bbox = union_geom.bounds
            from shapely.geometry import box, mapping
            bbox_geom = mapping(box(*bbox))
        except Exception as e:
            logging.error(f"Invalid GeoJSON features: {e}")
            context.set_code(grpc.StatusCode.INVALID_ARGUMENT)
            context.set_details(f"Invalid GeoJSON features: {e}")
            return image_service_pb2.ListAvailableDatesResponse()
        date_range = f"{request.start_date}/{request.end_date}"
        logging.info(f"Searching scenes with bbox={bbox} and date_range={date_range}")
        scenes = search_scenes(bbox_geom, date_range)
        available_dates = []
        for scene in scenes:
            band_keys = [BAND_MAP.get(band, {}).get("asset_key") for band in request.bands]
            if all(asset_key in scene.assets for asset_key in band_keys if asset_key):
                date = scene.properties.get("datetime", "")[:10]
                available_dates.append(date)
        logging.info(f"Available dates found: {available_dates}")
        return image_service_pb2.ListAvailableDatesResponse(available_dates=available_dates)

    def GetBandValues(self, request, context):
        logging.info(f"GetBandValues called with date={request.date}, bands={request.bands}, geojson_features={request.geojson_features}")
        try:
            geometries = [json.loads(g) for g in request.geojson_features]
            shapes = [shape(geom) for geom in geometries]
            from shapely.ops import unary_union
            union_geom = unary_union(shapes)
            bbox = union_geom.bounds
            from shapely.geometry import box, mapping
            bbox_geom = mapping(box(*bbox))
        except Exception as e:
            logging.error(f"Invalid GeoJSON features: {e}")
            context.set_code(grpc.StatusCode.INVALID_ARGUMENT)
            context.set_details(f"Invalid GeoJSON features: {e}")
            return image_service_pb2.GetBandValuesResponse()
        date_range = f"{request.date}/{request.date}"
        logging.info(f"Searching scenes with bbox={bbox} and date_range={date_range}")
        scenes = search_scenes(bbox_geom, date_range)
        if not scenes:
            logging.warning('No scene found for the given date')
            context.set_code(grpc.StatusCode.NOT_FOUND)
            context.set_details('No scene found for the given date')
            return image_service_pb2.GetBandValuesResponse()
        scene = scenes[0]
        band_data = {}  # map: band -> map: geometry_index -> bytes
        s3 = boto3.client('s3', region_name='us-west-2')
        scene_id = scene.id
        parts = scene_id.split('_')
        if len(parts) >= 4:
            tile_info = parts[1]
            date_part = parts[2]
            processing_level = parts[3]
            if len(tile_info) >= 5:
                utm_zone = tile_info[:2]
                lat_band = tile_info[2]
                grid_square = tile_info[3:5]
                year = date_part[:4]
                month = date_part[4:6]
                day = date_part[6:8]
                s3_prefix = f"tiles/{utm_zone}/{lat_band}/{grid_square}/{year}/{int(month)}/{int(day)}/{processing_level}/"
                for band in request.bands:
                    asset_info = BAND_MAP.get(band)
                    if not asset_info:
                        logging.warning(f"Band {band} not found in BAND_MAP")
                        continue
                    asset_key = asset_info["asset_key"]
                    resolution = asset_info["resolution"]
                    file_name = asset_info["file_name"]
                    asset = scene.assets.get(asset_key)
                    if asset is None:
                        logging.warning(f"Asset key {asset_key} not found in scene assets for band {band}")
                        continue
                    file_path = f"{s3_prefix}{resolution}/{file_name}"
                    try:
                        logging.info(f"Checking S3 for file: {file_path}")
                        s3.head_object(Bucket=S3_BUCKET, Key=file_path)
                        with tempfile.NamedTemporaryFile(suffix=".jp2") as tmp_jp2:
                            download_band(S3_BUCKET, file_path, tmp_jp2.name)
                            for idx, geom in enumerate(geometries):
                                try:
                                    cropped, meta = crop_band(tmp_jp2.name, geom)
                                    if cropped is None:
                                        logging.warning(f"Cropped band is None for band {band} geometry {idx}")
                                        continue
                                    with tempfile.NamedTemporaryFile(suffix=".tif") as tmp_tif:
                                        with rasterio.open(tmp_tif.name, "w", **meta) as dst:
                                            dst.write(cropped)
                                        tmp_tif.seek(0)
                                        if band not in band_data:
                                            band_data[band] = {}
                                        band_data[band][str(idx)] = tmp_tif.read()
                                except Exception as e:
                                    logging.error(f"Error cropping band {band} for geometry {idx}: {e}")
                                    continue
                    except Exception as e:
                        logging.error(f"Error processing band {band}: {e}")
                        continue
        logging.info(f"Returning band data for bands: {list(band_data.keys())}")
        # Flatten band_data to a map<string, bytes> with keys like band_geomidx
        flat_band_data = {}
        for band, geom_map in band_data.items():
            for idx, data in geom_map.items():
                flat_band_data[f"{band}_geom{idx}"] = data
        return image_service_pb2.GetBandValuesResponse(band_data=flat_band_data)

def serve():
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    image_service_pb2_grpc.add_ImageServiceServicer_to_server(ImageServiceServicer(), server)
    server.add_insecure_port('[::]:50051')
    server.start()
    print("ImageService gRPC server started on port 50051")
    server.wait_for_termination()

if __name__ == "__main__":
    serve() 