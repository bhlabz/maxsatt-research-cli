import json

def flatten_coordinates(coordinates):
    flat_coordinates = []
    for coord in coordinates:
        if isinstance(coord[0], list):
            flat_coordinates.extend(flatten_coordinates(coord))
        else:
            flat_coordinates.append(coord)
    return flat_coordinates

def get_centroid_latitude_longitude(geometry):
    flat_coordinates = flatten_coordinates(geometry['coordinates'])
    
    latitude = 0
    longitude = 0
    for coordinate in flat_coordinates:
        latitude += coordinate[1]
        longitude += coordinate[0]
    
    latitude /= len(flat_coordinates)
    longitude /= len(flat_coordinates)
    
    return latitude, longitude

def get_geometry_from_geojson(farm,plot):
    with open(f'geojsons/{farm}.geojson') as f:
        geojson = json.load(f)
    geometry = None
    for feature in geojson['features']:
        if feature['properties']['plot_id'] == plot:
            geometry = feature['geometry']
            break
    if geometry is None:
        raise Exception(f"Geometry not found for farm {farm} and plot {plot}")
    return geometry

def get_all_plots_and_geometries(farm):
    with open(f'geojsons/{farm}.geojson') as f:
        geojson = json.load(f)
    plots_and_geometries = []
    for feature in geojson['features']:
        plot = feature['properties']['plot_id']
        geometry = feature['geometry']
        plots_and_geometries.append((plot, geometry))
    return plots_and_geometries