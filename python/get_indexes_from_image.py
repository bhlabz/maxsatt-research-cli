import rasterio
import numpy as np
from io import BytesIO

def get_indexes_from_image(image):
    if isinstance(image, BytesIO):
        dataset = rasterio.open(image)
    else:
        dataset = rasterio.open(BytesIO(image))
    
    with dataset:
        bands = {
            "B05": dataset.read(1),  # Band 1: B05
            "B08": dataset.read(2),  # Band 2: B08
            "B11": dataset.read(3),  # Band 3: B11
            "B02": dataset.read(4),  # Band 4: B02
            "B04": dataset.read(5),  # Band 5: B04
            "B06": dataset.read(6),  # Band 6: B06
            "CLD": dataset.read(7),  # Band 7: Cloud probability (CLD)
            "SCL": dataset.read(8)   # Band 8: Scene classification (SCL)
        }
        
        # Calculate indexes
        ndre = np.divide((bands["B08"] - bands["B05"]), (bands["B08"] + bands["B05"]), out=np.zeros_like(bands["B08"]), where=(bands["B08"] + bands["B05"])!=0)
        ndmi = np.divide((bands["B08"] - bands["B11"]), (bands["B08"] + bands["B11"]), out=np.zeros_like(bands["B08"]), where=(bands["B08"] + bands["B11"])!=0)
        psri = np.divide((bands["B04"] - bands["B02"]), bands["B06"], out=np.zeros_like(bands["B04"]), where=bands["B06"]!=0)
        ndvi = np.divide((bands["B08"] - bands["B04"]), (bands["B08"] + bands["B04"]), out=np.zeros_like(bands["B08"]), where=(bands["B08"] + bands["B04"])!=0)
        
        indexes = {
            'ndre': ndre,
            'ndmi': ndmi,
            'psri': psri,
            'ndvi': ndvi,
            'b02': bands["B02"],
            'b04': bands["B04"],
            'cloud': bands["CLD"],
            'scl': bands["SCL"]
        }
    
    return indexes

def validate_indexes(psri_value, ndvi_value, ndmi_value, ndre_value, cld_value, scl_value, b02_value, b04_value, weather):
    return  not (np.isnan(psri_value) or np.isnan(ndvi_value) or np.isnan(ndmi_value) or np.isnan(ndre_value) or cld_value > 0 or scl_value in [3, 8, 9, 10] or (b04_value + b02_value) / 2 > 0.9 or weather['precipitation'] is None or weather['temperature'] is None or (psri_value == 0 and ndvi_value == 0 and  ndmi_value == 0 and ndre_value == 0))