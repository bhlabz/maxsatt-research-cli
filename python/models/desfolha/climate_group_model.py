import pandas as pd
from sklearn.preprocessing import StandardScaler
from sklearn.cluster import KMeans
import time


def select_columns(data, columns):
    df = pd.DataFrame(data)
    selected_data = df[columns]
    scaler = StandardScaler()
    scaled_data = scaler.fit_transform(selected_data)
    return pd.DataFrame(scaled_data, columns=columns, index=df.index)

def apply_kmeans(data, n_clusters):
    columns = [
        'avg_temperature_30_days', 'temp_std_dev_30_days', 'avg_humidity_30_days',
        'humidity_anomaly_30_days', 'humidity_std_dev_30_days', 'total_precipitation_30_days',
        'precipitation_anomaly_30_days', 'dry_days_consecutive_30_days'
    ]

    data_with_selected_columns = select_columns(data, columns)
    kmeans = KMeans(n_clusters=n_clusters, random_state=42)
    clusters = kmeans.fit_predict(data_with_selected_columns)
    return clusters

def retrieve_input_cluster_data(df):
    cluster_label = df.loc[df['label'].isnull(), 'cluster'].values[0]
    
    # Retrieve all data points in the same cluster
    same_cluster_data = df[df['cluster'] == cluster_label]
    
    return df.loc[df['label'].isnull()], same_cluster_data

def climate_group_model(data,n_clusters=8):

    clusters = apply_kmeans(data, n_clusters)

    data['cluster'] = clusters

    _, same_cluster_data = retrieve_input_cluster_data(data)

    return same_cluster_data
