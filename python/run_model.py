from models.desfolha.climate_group_model import climate_group_model
from models.desfolha.reflectance_model import reflectance_model
import pandas as pd
from get_weather import fetch_weather
from delta_dataset import create_delta_dataset
import json
from image_coordinate import xy_to_latlon
import os
from telegram import send_error_message, send_success_message,send_warn_message
from datetime import timedelta, datetime
from get_images import get_images
from create_dataset import get_climate_group_data
from geojson import get_geometry_from_geojson, get_all_plots_and_geometries
from tqdm import tqdm
import sys

def parse_result_to_infestation(infestation,result, date):
    infestation[date]=[]
    for sample in result:
        pest_prob = sum(value for key, value in sample.items() if key != "Saudavel" and key != 'x' and key != 'y')
        if pest_prob > 0.66:
            sample["pest_prob"] = pest_prob
            infestation[date].append(sample)
    return infestation


def generate_geojson_features(tiff_file_path,result,plot_id,name=None):

    for file in os.listdir(tiff_file_path):
        if file.endswith(".tif"):
            tiff_file_path = os.path.join(tiff_file_path, file)
            break

    features = []
    for sample in result:
        if any(value > 0.66 for key, value in sample.items() if key not in ["x", "y"]):
            feature = {
                "type": "Feature",
                "geometry": {
                    "type": "Point",
                    "coordinates": xy_to_latlon(tiff_file_path,sample["x"], sample["y"])
                },
                "properties": {"image_location":{"x":sample["x"], "y":sample["y"]}, "classification": {}, "plot_id": str(plot_id)}
            }
            for key, value in sample.items():
                if key not in ["x", "y"]:
                    feature["properties"]["classification"][key] = value

            features.append(feature)
    if name is not None:
        with open(f'results/{name}.geojson', 'w') as file:
            json.dump({
                "type": "FeatureCollection",
                "features": features
            }, file, indent=4)
    return features

def run_model(input,dataset, climate_group_clusters=2, reflectance_clusters=16):

    dataset_concat = pd.concat([dataset, input], ignore_index=True)

    result = climate_group_model(dataset_concat, climate_group_clusters)
    if len(result['label'].unique()) == 1:
        print("Only one cluster was found, skipping reflectance model")
        send_warn_message("Only one cluster was found, skipping reflectance model")
        return 
    result = reflectance_model(result, reflectance_clusters)

    return result
    
    # except Exception as e:
    #     send_error_message(e, f"Error processing desfolha model for {name}")
    
def get_dataset():
    dataset_path = 'data/climate_group/166.csv'
    dataset = pd.read_csv(dataset_path)
    return dataset

def evaluate_plot(farm,plot,dates):
    # farm = "Boi Preto XI"
    # plot = "024"
    # dates = ["2024-10-01"]
    delta_days = 5
    delta_days_trash_hold = 20
    infestation = {}

    for date in dates:
        end_date = datetime.strptime(date, '%Y-%m-%d')
        start_date = end_date - timedelta(days=delta_days+delta_days_trash_hold)
        
        geometry = get_geometry_from_geojson(farm,plot)

        images = get_images(geometry, farm, plot, start_date, end_date,1)

        historical_weather = fetch_weather(geometry, start_date - timedelta(days=4*30), end_date)

        delta_dataset = create_delta_dataset(images, historical_weather, delta_days, delta_days_trash_hold)

        climate_group = get_climate_group_data(delta_dataset, historical_weather, date, farm, plot, delta_days, delta_days_trash_hold, cache=False)
        
        dataset = get_dataset()
        
        result = run_model(
            climate_group,
            dataset
        )

        file_name = f"{farm}_{plot}_{date}_climate_group_clusters"
        generate_geojson_features(f"images/{farm}_{plot}", result, file_name)
        infestation = parse_result_to_infestation(infestation, result, date)

def evaluate_forest(farm,dates):
    # farm = "Boi Preto XI"
    # dates = ["2024-10-01"]
    delta_days = 5
    delta_days_trash_hold = 20


    for date in dates:
        end_date = datetime.strptime(date, '%Y-%m-%d')
        start_date = end_date - timedelta(days=delta_days+delta_days_trash_hold)
        
        plots_and_geometries = list(get_all_plots_and_geometries(farm))
        for plot_geometry in tqdm(plots_and_geometries, desc="Processing plots"):
            plot, geometry = plot_geometry
            try:
                images = get_images(geometry, farm, plot, start_date, end_date, 1)

                historical_weather = fetch_weather(geometry, start_date - timedelta(days=4*30), end_date)

                delta_dataset = create_delta_dataset(images, historical_weather, delta_days, delta_days_trash_hold)

                climate_group = get_climate_group_data(delta_dataset, historical_weather, date, farm, plot, delta_days, delta_days_trash_hold, cache=False)
                
                dataset = get_dataset()
                
                result = run_model(
                    climate_group,
                    dataset
                )

                features = generate_geojson_features(f"images/{farm}_{plot}", result, plot_id=plot)
                file_name = f"results/{farm}_forest_{date}_forest_heat_map.geojson"
                if os.path.exists(file_name):
                    with open(file_name, 'r') as file:
                        existing_data = json.load(file)
                    existing_data["features"].extend(features)
                    with open(file_name, 'w') as file:
                        json.dump(existing_data, file, indent=4)
                else:
                    with open(file_name, 'w') as file:
                        json.dump({
                            "type": "FeatureCollection",
                            "features": features
                        }, file, indent=4)
            except Exception as e:
                send_error_message(e, f"Error processing desfolha model for {farm} - {plot}")

    send_success_message(f"Desfolha model processed for {farm}")

if __name__ == '__main__':
    if len(sys.argv) != 4:
        print("Usage: python run_model.py <farm> <plot> <date>")
        sys.exit(1)
    farm = sys.argv[1]
    plot = sys.argv[2]
    date = sys.argv[3]

    evaluate_plot(farm, plot, [date])