from datetime import timedelta,datetime
from create_climate_dataset import get_dates_weather_dataset
from create_climate_group_dataset import create_climate_group_dataset
import os
import pandas as pd

def is_between_dates(date, start_date, end_date):
    date = pd.to_datetime(date)
    start_date = pd.to_datetime(start_date)
    end_date = pd.to_datetime(end_date)
    return start_date <= date <= end_date

def parse_date(date):
    if isinstance(date, datetime):
        return date
    return datetime.strptime(date, '%Y-%m-%d')


def get_climate_group_data(delta_dataset, historical_weather, date, farm, plot, delta_days=5, delta_days_trash_hold=3, cache=True, file_name=None):

    date = parse_date(date)
    name = farm+"_"+plot+"_"+date.strftime('%Y-%m-%d')
    if file_name is None:
        file_name = f"{name}.csv"

    if cache:
        file_path = os.path.join("data/climate_group", file_name)
        if os.path.exists(file_path):
            return pd.read_csv(file_path)

    end_date = date
    start_date = end_date - timedelta(days=delta_days+delta_days_trash_hold)

    delta_dataset = delta_dataset[
        ((pd.to_datetime(delta_dataset['start_date']) >= start_date) & 
        (pd.to_datetime(delta_dataset['start_date']) <= end_date)) |
        ((pd.to_datetime(delta_dataset['end_date']) >= start_date) & 
        (pd.to_datetime(delta_dataset['end_date']) <= end_date))
    ]

    dates = delta_dataset['end_date']
    climate_dataset = get_dates_weather_dataset(dates, historical_weather)
    
    return create_climate_group_dataset(delta_dataset, climate_dataset, file_name)