import numpy as np
from pygam import LinearGAM, s


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

def clear_and_smooth(data):
    smoothed_data = gam_smoothing(detect_outliers(data), 0.0005)
    return smoothed_data
