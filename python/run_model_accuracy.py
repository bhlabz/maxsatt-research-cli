from get_weather import fetch_weather
from delta_dataset import create_delta_dataset
import numpy as np
from telegram import send_success_message, send_error_message
from datetime import timedelta, datetime
from get_images import get_images
from create_dataset import get_climate_group_data
from geojson import get_geometry_from_geojson
from tqdm import tqdm
import concurrent.futures
from run_model import get_dataset, run_model
import os
import pandas as pd
import traceback

climate_group_clusters=6

def evaluate_model_accuracy(labeled_rows, climate_group, dataset):
    
    # Executar o modelo de clusterização
    result = run_model(climate_group, dataset,climate_group_clusters=climate_group_clusters)
    res = []
    for labeled_row in labeled_rows:
        for sample in result:
            prediction = {key: value for key, value in sample.items() if key not in ['x', 'y']}
            accurate = max(prediction, key=prediction.get) == labeled_row['label']
            res.append([prediction, labeled_row['label'], accurate])
    accuracy_percentage = sum(1 for _, _, accurate in res if accurate) / len(res) * 100
    return res, accuracy_percentage

def process_plot(farm, plot, rows):
    # try:
        labeled_rows = []
        for row in rows:
            labeled_row = row.copy()
            labeled_rows.append(labeled_row)
            del row['label']
        start_date = datetime.strptime(rows[0]['start_date'], '%Y-%m-%d')
        delta_days = 5
        delta_days_trash_hold = 20
        end_date = start_date + timedelta(days=delta_days + delta_days_trash_hold)

        climate_group_name = farm + "_" + plot + "_" + start_date.strftime('%Y-%m-%d') + ".csv"
        climate_group_path = os.path.join("data", "climate_group", climate_group_name)

        if os.path.exists(climate_group_path):
            climate_group = pd.read_csv(climate_group_path)
        else:
            geometry = get_geometry_from_geojson(farm, plot)
            images = get_images(geometry, farm, plot, start_date, end_date, 1)
            historical_weather = fetch_weather(geometry, start_date - timedelta(days=4*30), end_date)
            delta_dataset = create_delta_dataset(images, historical_weather, delta_days, delta_days_trash_hold)
            climate_group = get_climate_group_data(delta_dataset, historical_weather, start_date, farm, plot, delta_days, delta_days_trash_hold, cache=False)

        result = evaluate_model_accuracy(labeled_rows, climate_group, dataset)
        
        accurate_count = sum(1 for _, _, accurate in result[0] if accurate)
        prediction_count = len(result[0])
            
        return accurate_count, prediction_count
    # except Exception as e:
    #     send_error_message(e, f"Error processing desfolha model for {farm}_{plot}")
    #     return 0, 0


if __name__ == '__main__':
    try: 
        dataset = get_dataset()
        grouped_data = {}
        for index, row in dataset.iterrows():
            farm = row['name'].split('_')[0]
            plot = row['name'].split('_')[1]

            if farm not in grouped_data:
                grouped_data[farm] = {}
            
            if plot not in grouped_data[farm]:
                grouped_data[farm][plot] = []
            
            grouped_data[farm][plot].append(row)

        # Sort 30% of the plots for validation
        validation_plots = {}
        for farm, plots in grouped_data.items():
            plot_keys = list(plots.keys())
            np.random.shuffle(plot_keys)
            validation_count = int(len(plot_keys) * 0.30)
            validation_plots[farm] = {plot: plots[plot] for plot in plot_keys[:validation_count]}
            
            # Remove validation plots from dataset
            for plot in validation_plots[farm]:
                dataset = dataset[dataset['name'] != f"{farm}_{plot}"]



        total_accurate = 0
        total_predictions = 0

        for farm, plots in tqdm(validation_plots.items(), desc="Farms"):
            for plot, rows in plots.items():
                accurate_count, prediction_count = process_plot(farm, plot, rows)
                total_accurate += accurate_count
                total_predictions += prediction_count

        print(f"Total Accurate: {total_accurate}")
        print(f"Total Predictions: {total_predictions}")
        send_success_message(f"Total Accurate: {total_accurate}\nTotal Predictions: {total_predictions}\nAccuracy: {total_accurate/total_predictions*100}%\nClimate Group Clusters = {climate_group_clusters}")
    except Exception as e:
        tb = traceback.format_exc()
        send_error_message(f"{e}\n{tb}", "Error processing desfolha model")
        print(tb)
        raise e