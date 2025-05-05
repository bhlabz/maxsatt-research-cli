import os

import pandas as pd
from climate_group_model import climate_group_model
from reflectance_model import reflectance_model


def run_model(input, climate_group_clusters=2, reflectance_clusters=16):
    root = os.getenv('ROOT_PATH', '')
    dataset = pd.read_csv(f'{root}/data/model/166.csv')
    dataset_concat = pd.concat([dataset, input], ignore_index=True)

    result = climate_group_model(dataset_concat, climate_group_clusters)
    if len(result['label'].unique()) == 1:
        print("Only one cluster was found, skipping reflectance model")
        return 
    result = reflectance_model(result, reflectance_clusters)
    return result
