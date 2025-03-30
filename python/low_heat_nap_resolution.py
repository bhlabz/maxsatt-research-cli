import json
from collections import defaultdict

def low_heat_nap_resolution(geojson, target_resolution, received_resolution=10):
    # Parse the input GeoJSON
    data = json.loads(geojson)
    
    # Group points by plot_id
    grouped_points = defaultdict(list)
    for feature in data['features']:
        plot_id = feature['properties']['plot_id']
        grouped_points[plot_id].append(feature)
    
    # Calculate the number of pixels per cell
    pixels_per_cell = target_resolution // received_resolution
    print(f"Pixels per cell: {pixels_per_cell}")
    
    # Adjust points for each plot_id
    new_features = []
    for plot_id, features in grouped_points.items():
        # Create a grid for the new resolution
        grid = defaultdict(list)
        for feature in features:
            x = feature['properties']['image_location']['x']
            y = feature['properties']['image_location']['y']
            new_x = x // pixels_per_cell
            new_y = y // pixels_per_cell
            grid[(new_x, new_y)].append(feature)
        
        # Average the points in each cell
        for (new_x, new_y), cell_features in grid.items():
            avg_feature = cell_features[0].copy()
            avg_0 = sum(f['geometry']['coordinates'][0] for f in cell_features) / len(cell_features)
            avg_1 = sum(f['geometry']['coordinates'][1] for f in cell_features) / len(cell_features)
            avg_feature['geometry']['coordinates'][0] = avg_0
            avg_feature['geometry']['coordinates'][1] = avg_1
            avg_classification = defaultdict(float)
            for feature in cell_features:
                for key, value in feature['properties']['classification'].items():
                    avg_classification[key] += value
            for key in avg_classification:
                avg_classification[key] /= len(cell_features)
            avg_feature['properties']['classification'] = dict(avg_classification)
            new_features.append(avg_feature)
    
    # Create new GeoJSON
    new_geojson = {
        "type": "FeatureCollection",
        "features": new_features
    }
    
    print(f"Number of features: {len(new_features)}")
    return json.dumps(new_geojson, indent=2)

# Example usage
path = "results/Boi Preto XI_forest_2024-10-01_forest_heat_map"
with open(path + ".geojson", 'r') as file:
    geojson = file.read()

target_resolution = 200
result_geojson = low_heat_nap_resolution(geojson, target_resolution)
with open(path + f"_resolution_{target_resolution}.geojson", 'w') as file:
    file.write(result_geojson)