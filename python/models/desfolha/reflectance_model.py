from sklearn.mixture import GaussianMixture
from sklearn.preprocessing import StandardScaler
import numpy as np

def retrieve_sample_probabilities(index, sample, cluster_probabilities, cluster_distribution):
    cluster_id = sample["cluster"]
    cluster_probability = cluster_probabilities[:,cluster_id][index]
    sample_probability = {}
    for label, label_probability in cluster_distribution[cluster_id].items():
        sample_probability[label] = cluster_probability * label_probability
    return sample_probability
    

def reflectance_model(df, n_components=2, reg_covar=1e-6):
    # Extract relevant columns
    columns = ['delta', 'ndre', 'ndmi', 'psri', 'ndvi', 'ndre_derivative', 'ndmi_derivative', 'psri_derivative', 'ndvi_derivative']
    data = df[columns]

    # Preprocess the data
    scaler = StandardScaler()
    data_scaled = scaler.fit_transform(data)
    cluster_probabilities = None
    # Try fitting the GMM model with decreasing n_components until it succeeds
    while n_components > 0:
        try:
            gmm = GaussianMixture(n_components=n_components, reg_covar=reg_covar, random_state=42)
            df.loc[:, 'cluster'] = gmm.fit_predict(data_scaled)
            cluster_probabilities = gmm.predict_proba(data_scaled)
            break  # Exit the loop if fitting is successful
        except:
            n_components -= 1
            if n_components == 0:
                raise ValueError("Fitting the mixture model failed for all n_components values.")

    
    cluster_distribution = get_cluster_distribution(df)
    results = []
    df.reset_index(drop=True, inplace=True)
    null_label_indices = df[df['label'].isnull()].index
    for index in null_label_indices:
        sample = df.loc[index]
        sample_probabilities = retrieve_sample_probabilities(index, sample, cluster_probabilities, cluster_distribution)
        sample_probabilities['x'] = int(sample['x'])
        sample_probabilities['y'] = int(sample['y'])
        results.append(sample_probabilities) 

    return results

def retrieve_input_cluster_data(df):
    cluster_label = df.loc[df['label'].isnull(), 'cluster'].values[0]
    
    # Retrieve all data points in the same cluster
    same_cluster_data = df[df['cluster'] == cluster_label]
    
    return df.loc[df['label'].isnull()], same_cluster_data

def get_cluster_distribution(df):
    # Get unique clusters
    clusters = df['cluster'].unique()
    
    # Initialize a dictionary to store the mapping
    cluster_to_label = {}
    
    # Iterate over each cluster
    for cluster in clusters:
        cluster_data = df[df['cluster'] == cluster]
        # Get the subset of data for the current cluster
        
        # Ignore rows with null labels
        cluster_data = cluster_data[cluster_data['label'].notnull()]
        
        # Find the most frequent label in the current cluster
        if not cluster_data.empty:
            label_counts = cluster_data['label'].value_counts() / len(cluster_data)
            cluster_to_label[cluster] = label_counts.to_dict()
        else:
            cluster_to_label[cluster] = {
                'Desconhecido':1
            }
        
    return cluster_to_label


