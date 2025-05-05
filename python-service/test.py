import csv

# Define the file path
file_path = '/Users/gabihert/Documents/Projects/forest-guardian/forest-guardian-api-poc/data/model/166.csv'
output_file_path = '/Users/gabihert/Documents/Projects/forest-guardian/forest-guardian-api-poc/data/model/166_cleaned.csv'

# Define the columns to check
columns = [
    'avg_temperature', 'temp_std_dev', 'avg_humidity',
    'humidity_std_dev', 'total_precipitation',
    'dry_days_consecutive'
]

# Open and process the file
with open(file_path, mode='r') as file:
    reader = csv.DictReader(file)
    header = reader.fieldnames

    # Write the cleaned data to a new file
    with open(output_file_path, mode='w', newline='') as output_file:
        writer = csv.DictWriter(output_file, fieldnames=header)
        writer.writeheader()

        for row in reader:
            valid = True
            for column in columns:
                value = row.get(column, "")
                if not value.replace('.', '', 1).isdigit():  # Check if value is non-numeric
                    valid = False
                    break
            if valid:
                writer.writerow(row)

print(f"Cleaned file saved to {output_file_path}")