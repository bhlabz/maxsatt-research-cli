import json

def filter_geojson_by_plot_id(input_file_path, plot_id):
    # Step 1: Read the GeoJSON file
    with open(input_file_path, 'r') as file:
        geojson_data = json.load(file)
    
    # Step 2: Parse the GeoJSON data
    features = geojson_data.get('features', [])
    
    # Step 3: Filter the features based on the specified plot_id
    filtered_features = [feature for feature in features if feature.get('properties', {}).get('plot_id') == plot_id]
    
    # Step 4: Create a new GeoJSON object with the filtered features
    new_geojson = {
        "type": "FeatureCollection",
        "features": filtered_features
    }
    
    # Step 5: Return the new GeoJSON object
    return new_geojson

# Example usage
input_file_path = 'results/Boi Preto XI_forest_2024-10-01_forest_heat_map.geojson'
plot_id = '055'
filtered_geojson = filter_geojson_by_plot_id(input_file_path, plot_id)

# Save the filtered GeoJSON to a new file
output_file_path = f'results/Boi Preto XI_forest_2024-10-01_forest_heat_map_plot_{plot_id}.geojson'
with open(output_file_path, 'w') as file:
    json.dump(filtered_geojson, file, indent=2)