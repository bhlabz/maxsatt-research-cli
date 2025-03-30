from sentinelhub import (
    CRS,
    bbox_to_dimensions,
)
from sentinelhub import Geometry
import json
from datetime import datetime, timedelta
from request_image import request_image
import os
import json
from create_pixel_dataset import get_values, are_indexes_valid, get_indexes_from_image
import rasterio
from io import BytesIO

def get_images(geometry, farm, plot, start_date, end_date, satellite_interval_days = 5):

  polygon = Geometry(geometry, CRS.WGS84)
  betsiboka_size = bbox_to_dimensions(polygon.bbox, resolution=10)
  width_pixels = betsiboka_size[0]
  height_pixels = betsiboka_size[1]

  images_not_found_file = 'images/images_not_found.json'
  if os.path.exists(images_not_found_file):
      try:
          with open(images_not_found_file, 'r') as f:
              images_not_found = json.load(f)
      except json.JSONDecodeError as e:
          print(f"Warning: {images_not_found_file} contains invalid JSON. Initializing as an empty list.")
          raise e
  else:
      images_not_found = []
  
  images = {}
  
  # print(f"Getting images from {start_date.strftime('%Y-%m-%d')} to {end_date.strftime('%Y-%m-%d')}")
  while start_date <= end_date:
    start_date_str = start_date.strftime("%Y-%m-%dT00:00:00Z")
    end_date_str = start_date.strftime("%Y-%m-%dT23:59:59Z")
    try:
      file_name = f"images/{farm}_{plot}/{farm}_{start_date.strftime("%Y-%m-%d")}.tif"

      if file_name in images_not_found:
        # print(f"Image {file_name} is in the not found list. Skipping request.")
        start_date += timedelta(days=1) 
        continue

      if os.path.exists(file_name):
        # print(f"Image {file_name} already exists. Skipping request.")
        start_date += timedelta(days=satellite_interval_days)
        with open(file_name, "rb") as f:
          images[start_date_str.split('T')[0]] = BytesIO(f.read())
        continue
    
      image = request_image(start_date_str, end_date_str, geometry, width_pixels, height_pixels)

      with rasterio.open(BytesIO(image)) as dataset:
          x_range = range(0, dataset.width)
          y_range = range(0, dataset.height)
          total_pixels = dataset.width * dataset.height
      indexes = get_indexes_from_image(image)
      count = 0
      for y in y_range:
        for x in x_range:  
          ndmi_value, cld_value, scl_value, ndre_value, psri_value, b02_value, b04_value, ndvi_value = get_values(indexes,x,y)
          if not are_indexes_valid(psri_value, ndvi_value, ndmi_value, ndre_value, cld_value, scl_value, b02_value, b04_value):
              count += 1
      if count == total_pixels:
        raise Exception("Image not found")

      if not os.path.exists('images'):
        os.makedirs('images')
      if not os.path.exists(f'images/{farm}_{plot}'):
        os.makedirs(f'images/{farm}_{plot}')
      with open(file_name, "wb") as f:
        f.write(image)
        images[start_date_str.split('T')[0]] = BytesIO(image)
        
      # print(f"Image saved as {file_name}")
    except Exception as e:
      if str(e) == "Image not found":
        # print(f"No image found for date {start_date.strftime('%Y-%m-%')}")
        images_not_found.append(file_name)
        with open(images_not_found_file, 'w') as f:
          json.dump(images_not_found, f)
      else:
        print(f"Error: {e}")
      start_date += timedelta(days=1)
      continue
    start_date += timedelta(days=satellite_interval_days)

  return images

  