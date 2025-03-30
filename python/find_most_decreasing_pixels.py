import pandas as pd

# Step 1: Read the CSV file
file_path = 'data/delta/lagarta_euca_05a15/delta_18_10.csv'
df = pd.read_csv(file_path)

# Step 2: Convert 'start_date' to datetime
df['start_date'] = pd.to_datetime(df['start_date'])

# Step 3: Extract month and year for grouping
df['month_year'] = df['start_date'].dt.to_period('M')

# Step 4: Define derivative columns
derivative_columns = ['ndre_derivative', 'ndvi_derivative']

# Step 5: Find the row with the minimum value for each derivative column per month
results = {}
for col in derivative_columns:
    results[col] = df.loc[df.groupby('month_year')[col].idxmin()]

# Step 6: Print the results

for col, result in results.items():
    print(f"Results for {col}:")
    print(result[['start_date','x', 'y']])
    print("\n")