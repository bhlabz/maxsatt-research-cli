import requests
from collections import defaultdict
from datetime import datetime
import numpy as np
from geojson import get_centroid_latitude_longitude
import time

def calculate_mean_humidity(hourly_data):
    time = hourly_data['time']
    humidity = hourly_data['relative_humidity_2m']
    
    daily_humidity = defaultdict(list)
    
    for i in range(len(time)):
        t = time[i]
        h = humidity[i]
        date = datetime.strptime(t, '%Y-%m-%dT%H:%M').strftime('%Y-%m-%d')
        daily_humidity[date].append(h)
    
    mean_humidity = {}
    for date, humidities in daily_humidity.items():
        if None in humidities:
            mean_humidity[date] = None
        else:
            mean_humidity[date] = np.mean(humidities)
    
    return mean_humidity

def fetch_weather(geometry, start_date, end_date, retries=5):
    latitude, longitude = get_centroid_latitude_longitude(geometry)
    
    url = "https://archive-api.open-meteo.com/v1/archive"
    params = {
        "latitude": latitude,
        "longitude": longitude,
        "start_date": start_date.strftime('%Y-%m-%d'),
        "end_date": end_date.strftime('%Y-%m-%d'),
        "daily": "temperature_2m_mean,precipitation_sum",
        "hourly":"relative_humidity_2m"
    }

    attempt = 0
    while attempt < retries:
        response = requests.get(url, params=params)
        if response.status_code == 200:
            data = response.json()
            data_parsed = {}
            times = data['daily']['time']
            temperatures = data['daily']['temperature_2m_mean']
            precipitations = data['daily']['precipitation_sum']
            humidity = calculate_mean_humidity(data['hourly'])

            for i in range(len(times)):
                date = times[i]
                data_parsed[date] = {
                    "temperature": temperatures[i],
                    "precipitation": precipitations[i],
                    "humidity": humidity[date]
                }
            return data_parsed
        else:
            print(f"Failed to retrieve data: {response.status_code}. Retrying... ({attempt + 1}/{retries})")
            time.sleep(10)
            attempt += 1
    
    print("Failed to retrieve data after multiple attempts.")
    raise Exception("Failed to retrieve data after multiple attempts.")

if __name__ == "__main__":
    fetch_weather(-23.5489, -46.6388, datetime(2024, 11, 29))