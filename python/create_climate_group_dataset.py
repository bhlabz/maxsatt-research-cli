import pandas as pd
from concurrent.futures import ThreadPoolExecutor, as_completed
from tqdm import tqdm
import os
def process_sample(sample, weather_df, delta_df):
    date = sample["start_date"]
    x = sample["x"]
    y = sample["y"]
    label = sample.get("label", None)
    
    # Filter the weather and delta data for the given date
    weather_filtered = weather_df[weather_df['date'] == date]
    delta_filtered = pd.DataFrame([d for d in delta_df if d['start_date'] == date and d['x'] == x and d['y'] == y])
    if delta_filtered.empty:
        return None

    # Merge the filtered data
    merged_df = pd.merge(weather_filtered, delta_filtered, left_on='date', right_on='start_date')
    if merged_df.empty:
        return None

    # Add the label column
    merged_df['label'] = label
    # Add the created_at column with the current timestamp
    merged_df['created_at'] = pd.Timestamp.now()
    return merged_df

def process_samples_in_parallel(samples, weather_df, delta_df):
    result_df = pd.DataFrame()
    skipped = 0
    target = len(samples)
    
    with ThreadPoolExecutor() as executor:
        futures = {executor.submit(process_sample, sample, weather_df, delta_df): sample for sample in samples}
        for count, future in enumerate(as_completed(futures), 1):
            sample = futures[future]
            merged_df = future.result()
            if merged_df is not None:
                result_df = pd.concat([result_df, merged_df], ignore_index=True)
            else:
                skipped += 1
    
    return result_df

def is_between_dates(date, start_date, end_date):
    date = pd.to_datetime(date)
    start_date = pd.to_datetime(start_date)
    end_date = pd.to_datetime(end_date)
    return start_date <= date <= end_date

def create_climate_group_dataset(samples, weather_df, output_file_name=None):
    # Initialize an empty DataFrame to store the results

    # Iterate over the samples
    target = len(samples)
    count = 0
    merged_df = pd.DataFrame()
    progress_bar = tqdm(total=target, desc=f"Merging delta samples with climate data to create climate group dataset")
    for _, sample in samples.iterrows():
        count += 1
        progress_bar.update(1)
        weather_row = weather_df[
            (pd.to_datetime(weather_df['date']) >= pd.to_datetime(sample['start_date'])) & 
            (pd.to_datetime(weather_df['date']) <= pd.to_datetime(sample['end_date']))
        ].head(1)  
        if weather_row.empty:
            raise Exception(f"weather not found for {sample['start_date']} {sample['end_date']} when creating climate group dataset")

        merged_row = pd.DataFrame([sample])
        weather_row_series = weather_row.iloc[0]
        for col in weather_df.columns:
            merged_row[col] = weather_row_series[col]
        
        # Append the merged row to the merged_df
        merged_df = pd.concat([merged_df, merged_row], ignore_index=True)

        merged_df['label'] = sample.get('label', None)
        # Add the created_at column with the current timestamp
        merged_df['created_at'] = pd.Timestamp.now()
        # Append the merged data to the result DataFrame
    progress_bar.close()
    
    # Define the output file path
    if output_file_name is None:
        return merged_df
    output_file = 'data/climate_group/' + output_file_name
    if os.path.exists(output_file):
        merged_df.to_csv(output_file, mode='a', header=False, index=False, float_format='%.15g')
    else:
        merged_df.to_csv(output_file, index=False, float_format='%.15g')
    return merged_df

# # date:(x,y,label)
# samples = {
# ("2024-10-01",40,  13,"formiga"),
# ("2024-10-01", 40,  15,"formiga"),
# ("2024-10-01", 40,  16,"formiga")
# }
# delta_df = pd.read_csv('data/delta/Fazenda_Embay_026/delta_8_5.csv')
# weather_df = pd.read_csv('data/weather/Fazenda_Embay_026/dataset.csv')
# create_climate_group_dataset(samples, delta_df, weather_df)

