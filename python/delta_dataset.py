import os
from datetime import datetime, timedelta
import pandas as pd
from tqdm import tqdm
from create_pixel_dataset import create_pixel_dataset
from clean_dataset import clean_dataset
from datetime import timedelta,datetime
import os
import pandas as pd

def parse_date(date_str):
    return datetime.strptime(date_str, '%Y-%m-%d')

def delta_dataset(delta_min, delta_max, clear_dataset):

    delta_dataset = []

    clear_dataset.sort(key=lambda row: parse_date(row['date']))
    grouped_pixels = {}
    for row in clear_dataset:
        key = (row['x'], row['y'])
        if key not in grouped_pixels:
            grouped_pixels[key] = []
        grouped_pixels[key].append(row)


    found = 0
    not_found = 0

    target = len(grouped_pixels)
    progress_bar = tqdm(total=target, desc=f"Creating delta dataset")
    for key, data in list(grouped_pixels.items()):
        i = 0
        if len(data) < 3:
            not_found += 1
            # print(f"Pixel {key} does not have enough data points. Skipping")
            del grouped_pixels[key]
            progress_bar.update(1)
            continue
        while i < len(data) - 1:
            start_date = parse_date(data[i]["date"])
            min_target_date = start_date + timedelta(days=delta_min)
            max_target_date = start_date + timedelta(days=delta_max)
            start_date_str = start_date.strftime('%Y-%m-%d')
            
            for j in range(i + 1, len(data)):
                date = parse_date(data[j]["date"])
                if date >= min_target_date and date <= max_target_date:
                    x, y = data[i]["x"], data[i]["y"]
                    ndre_value = float(data[j]["ndre"]) - float(data[i]["ndre"])
                    ndmi_value = float(data[j]["ndmi"]) - float(data[i]["ndmi"])
                    psri_value = float(data[j]["psri"]) - float(data[i]["psri"])
                    ndvi_value = float(data[j]["ndvi"]) - float(data[i]["ndvi"])
                    
                    time_diff = (date - start_date).days
                    ndre_derivative = ndre_value / time_diff
                    ndmi_derivative = ndmi_value / time_diff
                    psri_derivative = psri_value / time_diff
                    ndvi_derivative = ndvi_value / time_diff

                    delta_dataset.append({
                        'delta_min': delta_min, 
                        'delta_max': delta_max, 
                        'delta': time_diff,
                        'start_date': start_date_str, 
                        'end_date': data[j]["date"], 
                        'x': x, 
                        'y': y, 
                        'ndre': ndre_value, 
                        'ndmi': ndmi_value, 
                        'psri': psri_value, 
                        'ndvi': ndvi_value,
                        'ndre_derivative': ndre_derivative,
                        'ndmi_derivative': ndmi_derivative,
                        'psri_derivative': psri_derivative,
                        'ndvi_derivative': ndvi_derivative
                    })
                    found += 1
                    break
                elif date > max_target_date:
                    not_found += 1
                    break
            i += 1
        progress_bar.update(1)

    # print(f"Found {found} deltas and {not_found} not found")

    if len(delta_dataset) == 0:
        raise ValueError("No valid delta data found. The delta_dataset is empty.")
    
    return pd.DataFrame(delta_dataset)

def create_delta_dataset(images, historical_weather, delta_days=5, delta_days_trash_hold=3):
    pixel_dataset =  create_pixel_dataset(images, historical_weather)
    clear_dataset = clean_dataset(pixel_dataset)
    delta = delta_dataset(delta_days,delta_days+delta_days_trash_hold, clear_dataset)
    return delta