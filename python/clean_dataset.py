import os
import matplotlib.pyplot as plt
import numpy as np
import csv
from pygam import LinearGAM, s
from tqdm import tqdm

def gam_smoothing(values, lam):
    x = np.arange(len(values)).reshape(-1, 1)
    y = np.array(values)
    
    if np.all(y == 0) or np.std(y) == 0:
        return y.tolist()

    gam = LinearGAM(s(0, lam=lam)).fit(x, y)
    y_pred = gam.predict(x)
    
    return y_pred.tolist()
    
def detect_outliers(data, window_size=5, threshold=0.3):
    """
    Detects and replaces outliers in a dataset using a sliding window approach.
    Parameters:
    data (list or array-like): The input dataset to be cleaned.
    window_size (int, optional): The size of the window to calculate the mean and standard deviation. Default is 5.
    threshold (float, optional): The threshold factor to determine if a data point is an outlier. Default is 0.3.
    Returns:
    list: A list containing the cleaned dataset with outliers replaced by the mean of the window.
    Example:
    >>> data = [1, 2, 1, 1, 100, 1, 2, 1, 1]
    >>> detect_outliers(data)
    [1, 2, 1, 1, 1.4, 1, 2, 1, 1]
    """
    cleaned_data = []
    for i in range(len(data)):
        start, end = max(0, i - window_size), min(len(data), i + window_size + 1)
        window = data[start:end]
        
        mean, std = np.mean(window), np.std(window)
        
        if abs(data[i] - mean) > threshold * std:
            cleaned_data.append(mean) 
        else:
            cleaned_data.append(data[i])

    return cleaned_data

smooth = 0.0005
def clean_dataset(pixel_dataset):
    grouped_data = {}
    for data in pixel_dataset:
        key = (data['x'], data['y'])
        if key not in grouped_data:
            grouped_data[key] = []
        grouped_data[key].append(data)

    progress_bar = tqdm(total=len(grouped_data), desc=f"Cleaning dataset")
    for key, data in grouped_data.items():
        ndre,ndmi,psri,ndvi = [],[], [], []
        for i in range(len(data)):
            ndre.append(float(data[i]['ndre']))
            ndmi.append(float(data[i]['ndmi']))
            psri.append(float(data[i]['psri']))
            ndvi.append(float(data[i]['ndvi']))


        ndmi = detect_outliers(ndmi)
        psri = detect_outliers(psri)
        ndre = detect_outliers(ndre)
        ndvi = detect_outliers(ndvi)

        ndvi = gam_smoothing(ndvi,smooth)
        ndmi = gam_smoothing(ndmi,smooth)
        psri = gam_smoothing(psri,smooth)
        ndre = gam_smoothing(ndre,smooth)
        
        valid_data = []
        for i in range(len(data)):
            ndmi_value = str(ndmi.pop(0))
            psri_value = str(psri.pop(0))
            ndre_value = str(ndre.pop(0))
            ndvi_value = str(ndvi.pop(0))
            if float(ndmi_value) == 0 or float(psri_value) == 0 or float(ndre_value) == 0 or float(ndvi_value) == 0:
                continue
            data[i]['ndmi'] = ndmi_value
            data[i]['psri'] = psri_value
            data[i]['ndre'] = ndre_value
            data[i]['ndvi'] = ndvi_value
            data[i]['precipitation'] = str(data[i]['precipitation'])
            data[i]['temperature'] = str(data[i]['temperature'])
            data[i]['humidity'] = str(data[i]['humidity'])
            valid_data.append(data[i])
        grouped_data[key] = valid_data
        progress_bar.update(1)

    new_array = []
    for key, data in grouped_data.items():
        for entry in data:
            new_array.append(entry)

    if len(new_array) > 0:
        return new_array 
    else:
        print(f"No valid data for")








if __name__ == "__main__":
    clean_dataset("data/not_clean", "boi_preto")