import os
import json
import rasterio
import boto3
import tempfile
from datetime import datetime
from rasterio.mask import mask
from shapely.geometry import shape, mapping
from shapely.ops import unary_union
from pystac_client import Client
from shapely.geometry import Polygon, MultiPolygon, box

# Config
BAND_MAP = {
    "B02": {"asset_key": "blue-jp2", "resolution": "R10m", "file_name": "B02.jp2"},
    "B04": {"asset_key": "red-jp2", "resolution": "R10m", "file_name": "B04.jp2"},
    "B05": {"asset_key": "rededge1-jp2", "resolution": "R20m", "file_name": "B05.jp2"},
    "B06": {"asset_key": "rededge2-jp2", "resolution": "R20m", "file_name": "B06.jp2"},
    "B08": {"asset_key": "nir-jp2", "resolution": "R10m", "file_name": "B08.jp2"},
    "B11": {"asset_key": "swir16-jp2", "resolution": "R20m", "file_name": "B11.jp2"},
    "SCL": {"asset_key": "scl-jp2", "resolution": "R20m", "file_name": "SCL.jp2"}
    # "CLD": not available in this collection
}
COLLECTION = "sentinel-2-l2a"
STAC_URL = "https://earth-search.aws.element84.com/v1"
S3_BUCKET = "sentinel-s2-l2a"

# Load polygon from GeoJSON
def load_polygons(geojson_path):
    with open(geojson_path) as f:
        geojson = json.load(f)
    return [feature["geometry"] for feature in geojson["features"]]

# Search STAC for relevant items
def search_scenes(geometry, date_range):
    client = Client.open(STAC_URL)
    search = client.search(
        collections=[COLLECTION],
        intersects=geometry,
        datetime=date_range,
        query={"eo:cloud_cover": {"lt": 40}}
    )
    return list(search.get_items())

# Download band from AWS S3
s3 = boto3.client('s3')
def download_band(bucket_name, key, dest):
    s3 = boto3.client('s3')

    # Ensure the file path is correct (for example, use one of the available paths like R10m, R20m, or R60m)
    print(f"Downloading from: {key}")

    try:
        s3.download_file(bucket_name, key, dest, ExtraArgs={"RequestPayer": "requester"})
        print(f"File downloaded successfully to {dest}")
    except Exception as e:
        print(f"Error: {e}")


# Crop a single band to the polygon
def crop_band(band_path, polygon):
    with rasterio.open(band_path) as src:
        # Debug: print raster bounds and CRS
        print(f"Raster bounds: {src.bounds}")
        print(f"Raster CRS: {src.crs}")
        print(f"Raster shape: {src.shape}")
        
        # Convert polygon to Shapely geometry if it's not already
        from shapely.geometry import shape, mapping
        from shapely.ops import transform
        import pyproj
        
        if isinstance(polygon, dict):
            poly_shape = shape(polygon)
        else:
            poly_shape = polygon
            
        print(f"Original polygon bounds: {poly_shape.bounds}")
        print(f"Original polygon area: {poly_shape.area}")
        
        # Transform polygon to raster's CRS
        transformer = pyproj.Transformer.from_crs(
            "EPSG:4326",  # WGS84
            src.crs,
            always_xy=True
        )
        poly_shape_transformed = transform(transformer.transform, poly_shape)
        
        print(f"Transformed polygon bounds: {poly_shape_transformed.bounds}")
        print(f"Transformed polygon area: {poly_shape_transformed.area}")
        
        # Remove Z dimension if present
        def _to_2d(geom):
            return transform(lambda x, y, z=None: (x, y), geom)
        poly_shape_2d = _to_2d(poly_shape_transformed)
        
        # Debug: check type and validity
        from shapely.geometry.base import BaseGeometry
        print("Type of poly_shape_2d:", type(poly_shape_2d))
        print("Is instance of BaseGeometry:", isinstance(poly_shape_2d, BaseGeometry))
        print("Is geometry valid:", poly_shape_2d.is_valid)
        print("GeoJSON mapping:", mapping(poly_shape_2d))
        
        # Handle MultiPolygon vs Polygon for rasterio.mask.mask
        if isinstance(poly_shape_2d, MultiPolygon):
            geoms = [mapping(p) for p in poly_shape_2d.geoms]
        else:
            geoms = [mapping(poly_shape_2d)]

        # Check if polygon intersects with raster using shapely box
        raster_bounds_geom = box(*src.bounds)
        if not poly_shape_2d.intersects(raster_bounds_geom):
            print("WARNING: Polygon does not intersect with raster bounds!")
            return None, None
            
        # Pass a GeoJSON-like dict to rasterio.mask.mask
        out_image, out_transform = mask(src, geoms, crop=True)
        out_meta = src.meta.copy()
        out_meta.update({
            "driver": "GTiff",
            "height": out_image.shape[1],
            "width": out_image.shape[2],
            "transform": out_transform
        })
        return out_image, out_meta
    

def list_s3_directory(bucket_name, prefix=""):
    s3 = boto3.client('s3')
    
    # List objects in the bucket with the provided prefix (if any)
    response = s3.list_objects_v2(Bucket=bucket_name, Prefix=prefix)

    if 'Contents' in response:
        print("Files in the directory:")
        for obj in response['Contents']:
            print(f" - {obj['Key']}")
    else:
        print("No files found.")
        
def process(geojson_path, date_range, output_dir):
    os.makedirs(output_dir, exist_ok=True)
    polygons = load_polygons(geojson_path)
    # Compute union bounding box
    shapes = [shape(geom) for geom in polygons]
    union_geom = unary_union(shapes)
    union_bounds = union_geom.bounds  # (minx, miny, maxx, maxy)
    # Create a bounding box geometry for search
    bbox_geom = mapping(box(*union_bounds))
    scenes = search_scenes(bbox_geom, date_range)

    if not scenes:
        print("No scenes found.")
        return

    for scene in scenes:
        scene_id = scene.id
        print(f"\nProcessing scene {scene_id}")
        print("Available asset keys:", list(scene.assets.keys()))
        parts = scene_id.split('_')
        if len(parts) >= 4:
            tile_info = parts[1]  # 22KBC
            date_part = parts[2]  # 20230629
            processing_level = parts[3]  # 0
            if len(tile_info) >= 5:
                utm_zone = tile_info[:2]  # 22
                lat_band = tile_info[2]   # K
                grid_square = tile_info[3:5]  # BC
                year = date_part[:4]
                month = date_part[4:6]
                day = date_part[6:8]
                s3_prefix = f"tiles/{utm_zone}/{lat_band}/{grid_square}/{year}/{int(month)}/{int(day)}/{processing_level}/"
                print(f"S3 prefix: {s3_prefix}")
                print("Checking available files in S3...")
                list_s3_directory(S3_BUCKET, s3_prefix)
                for band, asset_info in BAND_MAP.items():
                    asset_key = asset_info["asset_key"]
                    resolution = asset_info["resolution"]
                    file_name = asset_info["file_name"]
                    asset = scene.assets.get(asset_key)
                    if asset is None:
                        print(f"Band {band} not available in scene {scene_id}")
                        continue
                    file_path = f"{s3_prefix}{resolution}/{file_name}"
                    print(f"Checking if file exists: {file_path}")
                    try:
                        s3.head_object(Bucket=S3_BUCKET, Key=file_path)
                        print(f"✓ File exists, downloading {band} ({file_name})")
                        with tempfile.NamedTemporaryFile(suffix=".jp2") as tmp_jp2:
                            download_band(S3_BUCKET, file_path, tmp_jp2.name)
                            # For each geometry, crop from the big image
                            for idx, polygon in enumerate(polygons):
                                try:
                                    cropped, meta = crop_band(tmp_jp2.name, polygon)
                                    if cropped is None:
                                        print(f"Skipping {band} - no intersection with raster for geometry {idx}")
                                        continue
                                except Exception as e:
                                    print(f"Error cropping {band} for geometry {idx}: {e}")
                                    continue
                                out_path = os.path.join(output_dir, f"{scene_id}_{band}_geom{idx}.tif")
                                with rasterio.open(out_path, "w", **meta) as dst:
                                    dst.write(cropped)
                                print(f"Saved {out_path}")
                    except Exception as e:
                        print(f"✗ File does not exist or not accessible: {file_path}")
                        print(f"  Error: {e}")
            else:
                print(f"Could not parse tile info from scene ID: {scene_id}")
        else:
            print(f"Could not parse scene ID: {scene_id}")

# Example usage
if __name__ == "__main__":
    # Uncomment to explore what's available in S3
    # list_s3_directory('sentinel-s2-l2a', 'tiles/22/K/BC/2023/6/29/0/')  
    process(
        geojson_path="../../data/geojsons/test.geojson",
        date_range="2023-05-01/2023-06-30",
        output_dir="output_tiffs"
    )