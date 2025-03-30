from get_weather import fetch_weather
import pandas as pd
from datetime import datetime, timedelta
import matplotlib.pyplot as plt
import json


def calculate_metrics(df, period_days, target_date, historical_df):
    metrics = {}
    avg_temp = df["temperature"].mean()
    temp_std = df["temperature"].std()
    # Calculate temperature anomaly
    temp_anomaly = avg_temp - historical_df.loc[
        (historical_df["date"] >= target_date - timedelta(days=period_days)) & 
        (historical_df["date"] < target_date), "temperature"
    ].mean()

    # Calculate average humidity, humidity anomaly, and humidity standard deviation
    avg_humidity = df["humidity"].mean()
    humidity_anomaly = avg_humidity - historical_df.loc[
        (historical_df["date"] >= target_date - timedelta(days=period_days)) & 
        (historical_df["date"] < target_date), "humidity"
    ].mean()
    humidity_std = df["humidity"].std()

    # Calculate total precipitation and precipitation anomaly
    total_precip = df["precipitation"].sum()
    precip_anomaly = total_precip - historical_df.loc[
        (historical_df["date"] >= target_date - timedelta(days=period_days)) & 
        (historical_df["date"] < target_date), "precipitation"
    ].sum()

    dry_days_consecutive = (
        (df["precipitation"] == 0).astype(int).groupby(df["precipitation"].ne(0).cumsum()).cumsum().max()
    )

    metrics[f"avg_temperature_{period_days}_days"] = avg_temp
    metrics[f"temp_std_dev_{period_days}_days"] = temp_std
    metrics[f"temp_anomaly_{period_days}_days"] = temp_anomaly
    metrics[f"avg_humidity_{period_days}_days"] = avg_humidity
    metrics[f"humidity_anomaly_{period_days}_days"] = humidity_anomaly
    metrics[f"humidity_std_dev_{period_days}_days"] = humidity_std
    metrics[f"total_precipitation_{period_days}_days"] = total_precip
    metrics[f"precipitation_anomaly_{period_days}_days"] = precip_anomaly
    metrics[f"dry_days_consecutive_{period_days}_days"] = dry_days_consecutive
    return metrics

def json_to_df(weather_data):
    data = []
    for d, values in weather_data.items():
        data.append({
            "date": datetime.strptime(d, "%Y-%m-%d"),
            "precipitation": values.get("precipitation", 0),
            "temperature": values.get("temperature", 0),
            "humidity": values.get("humidity", 0),
        })
    return pd.DataFrame(data)

def get_weather_dataset_from_to(latitude, longitude, start_date, end_date, interval, historical_weather):
    # Converte o JSON em DataFrame
    df_historical = json_to_df(historical_weather)

    # Cria um DataFrame vazio para armazenar os resultados
    df_combined = pd.DataFrame()

    # Itera sobre as datas-alvo para obter os dados climáticos
    dates = []
    current_date = start_date
    while current_date <= end_date:
        dates.append(current_date.strftime("%Y-%m-%d"))        
        current_date += interval

    return get_dates_weather_dataset(latitude,longitude,dates,historical_weather)

def get_dates_weather_dataset(dates, historical_weather):
    df_historical = json_to_df(historical_weather)

    df_combined = pd.DataFrame()

    # print(f"Processing weather data for {latitude}, {longitude}")
    for date in dates:
        climate_data = get_location_weather_sample(date, df_historical)
        df_new = pd.DataFrame([climate_data])
        df_combined = pd.concat([df_combined, df_new], ignore_index=True)

    return df_combined

def get_location_weather_sample(date, df_historical):
    # Converte a data fornecida para o formato datetime
    target_date = datetime.strptime(date, "%Y-%m-%d")

    # Define o início dos períodos necessários
    start_date_4m = (target_date - timedelta(days=120)).strftime("%Y-%m-%d")
    start_date_1m = (target_date - timedelta(days=30)).strftime("%Y-%m-%d")

    # Cria recortes para os períodos de 1 mês e 4 meses
    df_30_days = df_historical[(df_historical["date"] >= pd.to_datetime(start_date_1m)) & 
                                (df_historical["date"] <= target_date)]
    # Processa os dados para calcular os parâmetros necessários


    metrics_30_days = calculate_metrics(df_30_days, 30, target_date, df_historical)

    # Combina os resultados
    final_metrics = {"date": target_date.strftime("%Y-%m-%d"), **metrics_30_days}
    return final_metrics


def plot_climate_data(climate_data):
    labels = list(climate_data.keys())[1:]  # Exclude the date
    values = list(climate_data.values())[1:]  # Exclude the date

    plt.figure(figsize=(10, 6))
    plt.bar(labels, values, color='skyblue')
    plt.xlabel('Metrics')
    plt.ylabel('Values')
    plt.title(f"Climate Data for {climate_data['date']}")
    plt.xticks(rotation=90)
    plt.tight_layout()
    plt.show()

def create_climate_dataset(dates, historical_weather,name=None):

    weather_data = get_dates_weather_dataset(dates, historical_weather)
    
    if name is not None:
        dataset_path = f'data/weather/{name}.csv'   
        weather_data.to_csv(dataset_path, index=False)
    return weather_data

if __name__ == "__main__":
    create_climate_dataset()