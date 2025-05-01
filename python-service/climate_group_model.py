import time

import pandas as pd
from sklearn.cluster import KMeans
from sklearn.preprocessing import StandardScaler


def select_columns(data, columns):
    df = pd.DataFrame(data)
    selected_data = df[columns]
    scaler = StandardScaler()
    scaled_data = scaler.fit_transform(selected_data)
    return pd.DataFrame(scaled_data, columns=columns, index=df.index)

def apply_kmeans(data, n_clusters):
    columns = [
        'avg_temperature', 'temp_std_dev', 'avg_humidity',
        'humidity_std_dev', 'total_precipitation',
        'dry_days_consecutive'
    ]

    data_with_selected_columns = select_columns(data, columns)
    kmeans = KMeans(n_clusters=n_clusters, random_state=42)
    clusters = kmeans.fit_predict(data_with_selected_columns)
    return clusters

def retrieve_input_cluster_data(df):
    cluster_label = df.loc[df['label'] == '', 'cluster'].values[0]
    
    # Retrieve all data points in the same cluster
    same_cluster_data = df[df['cluster'] == cluster_label]
    
    return df.loc[df['label'] == ''], same_cluster_data

def climate_group_model(data,n_clusters=8):

    clusters = apply_kmeans(data, n_clusters)

    data['cluster'] = clusters

    _, same_cluster_data = retrieve_input_cluster_data(data)

    return same_cluster_data
