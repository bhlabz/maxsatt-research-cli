from get_images import get_images
from create_dataset import get_climate_group_data
from delta_dataset import create_delta_dataset
from get_weather import fetch_weather
import json
from datetime import timedelta
import csv
import pandas as pd
from tqdm import tqdm
import traceback
from telegram import send_success_message,send_error_message

formiga_alto = 3
formiga_baixo = 1
formiga_medio = 2

psilideo_alto = 5
psilideo_baixo = 1
psilideo_medio = 2

lagarta_alto = 3
lagarta_baixo = 1
lagarta_medio = 2

percevejo_alto = 3
percevejo_baixo = 1
percevejo_medio = 2

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

def get_samples_amount_from_severity(severity, total_samples):
    if severity == "Alto":
        return int(0.15 * total_samples)
    elif severity == "Baixo":
        return int(0.05 * total_samples)
    else:
        return int(0.10 * total_samples)


def get_best_samples_from_delta_dataset(delta_dataset, samples_amount, name, label):
    
    # Sort the rows based on the lowest values of the specified derivatives
    sorted_rows = delta_dataset.sort_values(
        by=['ndre_derivative', 'ndmi_derivative', 'ndvi_derivative', 'psri_derivative'],
        ascending=[True, True, True, False]
    )
    
    # Add name and label columns
    sorted_rows['name'] = name
    sorted_rows['label'] = label
    
    # Select the top samples_amount rows
    best_samples = sorted_rows.head(samples_amount)
    
    return best_samples


def get_days_before_evidence_to_analyse(pest, severity):
    if pest == "Psilideo":
        if severity == "Alto":
            return psilideo_alto
        elif severity == "Baixo":
            return psilideo_baixo
        else:
            return psilideo_medio
    elif pest == "Formiga":
        if severity == "Alto":
            return formiga_alto
        elif severity == "Baixo":
            return formiga_baixo
        else:
            return formiga_medio
    elif pest == "Lagarta Desfolhadora":
        if severity == "Alto":
            return lagarta_alto
        elif severity == "Baixo":
            return lagarta_baixo
        else:
            return lagarta_medio
    elif pest == "Percevejo Bronzeado":
        if severity == "Alto":
            return percevejo_alto
        elif severity == "Baixo":
            return percevejo_baixo
        else:
            return percevejo_medio

def run_create_dataset():
    errors = []
    days_before_evidence_to_analyse = 5
    delta_days = 5
    delta_days_trash_hold = 20
    days_to_fetch = delta_days + delta_days_trash_hold + days_before_evidence_to_analyse

    output_file_name = f"166.csv"

    validation_data_path = 'validations/166.csv'
    with open(validation_data_path, 'r') as f:
        reader = csv.DictReader(f, delimiter=';')
        rows = list(reader)
        df = pd.read_csv(validation_data_path, delimiter=';')
        rows = df.to_dict(orient='records')
    
    name = "unknown"
    target = len(rows)
    progress_bar = tqdm(total=target, desc=f"Creating dataset from file {validation_data_path}")
    for i in range(target):
        try:
            date = pd.to_datetime(rows[i]['date'], format='%d/%m/%y')
            pest = rows[i]['pest']
            severity = rows[i]['severity']
            farm = rows[i]['farm']
            plot = rows[i]['plot'].split("-")[1]
            name = farm+"_"+plot+"_"+date.strftime('%Y-%m-%d')

            days_before_evidence_to_analyse = - get_days_before_evidence_to_analyse(pest,severity)
            geometry = get_geometry_from_geojson(farm,plot)
            

            end_date = date - timedelta(days=days_before_evidence_to_analyse-5) 
            start_date = end_date - timedelta(days=days_to_fetch)

            images = get_images(geometry, farm, plot, start_date, end_date,1)
            historical_weather = fetch_weather(geometry, start_date - timedelta(days=4*30), end_date)
            
            delta_dataset = create_delta_dataset(images, historical_weather, delta_days, delta_days_trash_hold)
            
            samples_amount = get_samples_amount_from_severity(severity, len(delta_dataset))
            best_samples = get_best_samples_from_delta_dataset(delta_dataset, samples_amount, name, pest)

            get_climate_group_data(best_samples, historical_weather, date, farm, plot, delta_days, delta_days_trash_hold, cache=False, file_name=output_file_name)
        except Exception as e:
            print(f"\n\n>>> Error processing {name}: \n{e}\n\n")
            errors.append(e)
            send_error_message(e, f"Error processing {name}")
            continue
        progress_bar.update(1)

        with open('errors.json', 'w') as error_file:
            error_details = []
            for error in errors:
                error_details.append({
                    'error': str(error),
                    'traceback': traceback.format_exception(None, error, error.__traceback__)
                })
            json.dump(error_details, error_file, indent=4)
    
if __name__ == "__main__":
    run_create_dataset()
    send_success_message("All samples processed successfully!")
