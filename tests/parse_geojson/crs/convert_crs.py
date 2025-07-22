import os
import json
from glob import glob
from shapely.geometry import shape, mapping
from shapely.ops import transform
from shapely.geometry.polygon import orient
from pyproj import Transformer

INPUT_DIR = os.path.join(os.path.dirname(__file__), 'input')
OUTPUT_DIR = os.path.join(os.path.dirname(__file__), 'output')
DEFAULT_SRC_CRS = 'EPSG:31982'
DST_CRS = 'EPSG:4326'  # OGC:CRS84 is equivalent to EPSG:4326 (WGS84 lon/lat)

os.makedirs(INPUT_DIR, exist_ok=True)
os.makedirs(OUTPUT_DIR, exist_ok=True)

def get_src_crs(crs_name):
    if crs_name in [
        'urn:ogc:def:crs:EPSG::31982',
        'EPSG:31982',
        'urn:ogc:def:crs:EPSG::5880',
        'EPSG:5880',
        'urn:ogc:def:crs:EPSG:31982',
        'urn:ogc:def:crs:EPSG:5880',
    ]:
        # Normalize to EPSG:31982 or EPSG:5880
        if '5880' in crs_name:
            return 'EPSG:5880'
        else:
            return 'EPSG:31982'
    return None

def reproject_geometry(geom, transformer):
    g = shape(geom)
    g = transform(transformer.transform, g)
    # Enforce right-hand rule for polygons and multipolygons
    if g.geom_type == 'Polygon':
        g = orient(g, sign=1.0)  # CCW for exterior
    elif g.geom_type == 'MultiPolygon':
        g = type(g)([orient(p, sign=1.0) for p in g.geoms])
    return g

def update_crs(geojson):
    geojson['crs'] = {
        "type": "name",
        "properties": {"name": "urn:ogc:def:crs:OGC:1.3:CRS84"}
    }
    for i, feature in enumerate(geojson.get('features', [])):
        feature['properties']['plot_id'] = i + 1
    return geojson

def process_geojson_file(input_path, output_path):
    with open(input_path, 'r', encoding='utf-8') as f:
        data = json.load(f)

    crs_name = data.get('crs', {}).get('properties', {}).get('name', '')
    if crs_name in ['urn:ogc:def:crs:OGC:1.3:CRS84', 'EPSG:4326']:
        # No transformation needed, just update CRS if necessary
        data = update_crs(data)
    else:
        src_crs = get_src_crs(crs_name) or DEFAULT_SRC_CRS
        transformer = Transformer.from_crs(src_crs, DST_CRS, always_xy=True)
        if 'features' in data:  # FeatureCollection
            for feature in data['features']:
                geom = feature['geometry']
                feature['geometry'] = mapping(reproject_geometry(geom, transformer))
            data = update_crs(data)
        elif 'geometry' in data:  # Single Feature
            data['geometry'] = mapping(reproject_geometry(data['geometry'], transformer))
            data = update_crs(data)
        else:
            print(f"Skipping {input_path}: not a valid GeoJSON Feature or FeatureCollection.")
            return

    with open(output_path, 'w', encoding='utf-8') as f:
        json.dump(data, f, ensure_ascii=False, indent=2)

if __name__ == '__main__':
    geojson_files = glob(os.path.join(INPUT_DIR, '*.geojson'))
    if not geojson_files:
        print(f"No GeoJSON files found in {INPUT_DIR}")
    for input_file in geojson_files:
        filename = os.path.basename(input_file)
        output_file = os.path.join(OUTPUT_DIR, filename)
        print(f"Processing {filename} ...")
        process_geojson_file(input_file, output_file)
    print("Done.") 