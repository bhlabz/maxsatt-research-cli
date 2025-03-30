import os
import matplotlib.pyplot as plt
import pandas as pd
import rasterio
from io import BytesIO
from get_indexes_from_image import get_indexes_from_image
import numpy as np
import csv
from tqdm import tqdm


def create_pixel_dataset(images, weather):
    """
    Creates a pixel dataset from a dictionary of images.
    Parameters:
    images (dict): A dictionary with the image date as key and the image as value (BytesIO).
    weather (dict): A dictionary containing weather data with dates as keys.
    Returns:
    list: A list of dictionaries containing pixel data.
    """

    width = 1
    height = 1
    total_pixels = 0
    
    x_range = range(0, width)
    y_range = range(0, height)

    for _, image_data in images.items():
        with rasterio.open(image_data) as dataset:
          if total_pixels == 0:
                x_range = range(0, dataset.width)
                y_range = range(0, dataset.height)
                width = dataset.width
                height = dataset.height
                total_pixels = dataset.width * dataset.height
                break
          
    file_results = []
    count = 0
    target = len(y_range) * len(x_range) * len(images)
    progress_bar = tqdm(total=target, desc=f"Creating pixel dataset")
    for y in y_range:
        for x in x_range:  
            sorted_image_dates = sorted(images.keys())
            for date in sorted_image_dates:
                image = images[date]

                result = get_data(image, total_pixels, width, height, x, y, date, weather[date])
                count += 1

                if result is not None:
                    file_results.append(result)

                progress_bar.update(1)
                
    if len(file_results) == 0:
        raise Exception("No data available to create the dataset")
    return file_results

def get_values(indexes,x,y):
    ndmi_value = indexes['ndmi'][y,x]
    cld_value= indexes['cloud'][y,x]
    scl_value= indexes['scl'][y,x]
    ndre_value = indexes['ndre'][y,x]
    psri_value = indexes['psri'][y,x]
    b02_value = indexes['b02'][y,x]
    b04_value = indexes['b04'][y,x]
    ndvi_value = indexes['ndvi'][y,x]
    return ndmi_value, cld_value, scl_value, ndre_value, psri_value, b02_value, b04_value, ndvi_value

def are_indexes_valid(psri_value, ndvi_value, ndmi_value, ndre_value, cld_value, scl_value, b02_value, b04_value):
    invalid_conditions = [
        (np.isnan(psri_value), "PSRI value is NaN"),
        (np.isnan(ndvi_value), "NDVI value is NaN"),
        (np.isnan(ndmi_value), "NDMI value is NaN"),
        (np.isnan(ndre_value), "NDRE value is NaN"),
        (cld_value > 0, "Cloud value is greater than 0"),
        (scl_value in [3, 8, 9, 10], "SCL value is in [3, 8, 9, 10]"),
        ((b04_value + b02_value) / 2 > 0.9, "(B04 value + B02 value) / 2 is greater than 0.9"),
        (psri_value == 0 and ndvi_value == 0 and ndmi_value == 0 and ndre_value == 0, "All index values are 0")
    ]

    for condition, reason in invalid_conditions:
        if condition:
            # print(f"Invalid condition: {reason}")
            return False

    return True

def is_weather_valid(weather):
    return (weather['precipitation'] is None or weather['temperature'] is None),
    
def get_data(image, total_pixels, width, height, x, y, date, weather=None):
    if total_pixels != 0 and total_pixels != width * height:
        raise Exception("Different image size")
    
    indexes = get_indexes_from_image(image)
    
    ndmi_value, cld_value, scl_value, ndre_value, psri_value, b02_value, b04_value, ndvi_value = get_values(indexes,x,y)
    
    if is_weather_valid(weather) and are_indexes_valid(psri_value, ndvi_value, ndmi_value, ndre_value, cld_value, scl_value, b02_value, b04_value):
        return {'date':date, 'x': x, 'y': y, 'ndre': ndre_value, 'ndmi': ndmi_value, 'psri': psri_value, 'ndvi': ndvi_value, 'temperature': weather['temperature'], 'precipitation': weather['precipitation'], 'humidity': weather['humidity']}
    return None

if __name__ == "__main__":
    create_pixel_dataset('images', 'boi_preto')