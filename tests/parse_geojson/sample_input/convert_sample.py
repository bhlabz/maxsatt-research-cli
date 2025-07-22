import csv
import json
import os
from datetime import datetime

INPUT_DIR = os.path.join(os.path.dirname(__file__), 'input')
OUTPUT_DIR = os.path.join(os.path.dirname(__file__), 'output')
OUTPUT_CSV = os.path.join(OUTPUT_DIR, 'features_summary.csv')

# Ensure output directory exists
os.makedirs(OUTPUT_DIR, exist_ok=True)

def get_pest_from_filename(filename):
    # Assumes pest is the last part after the last underscore and before .geojson
    base = os.path.basename(filename)
    name, _ = os.path.splitext(base)
    parts = name.split('-')
    return parts[-1] if len(parts) > 1 else ''

def process_geojson_file(filepath):
    serial = 1
    forest = os.path.splitext(os.path.basename(filepath))[0]
    pest = get_pest_from_filename(filepath)
    with open(filepath, 'r', encoding='utf-8') as f:
        data = json.load(f)
    features = data.get('features', [])
    rows = []
    serial = 1
    for feature in features:
        serial += 1
        props = feature.get('properties', {})
        plot = props.get('plot_id', serial)
        severity = props.get('severity', 'LOW')
        date = props.get('date', '')
        rows.append({
            'forest': forest,
            'plot': plot,
            'pest': pest,
            'severity': severity,
            'date': date
        })
    return rows

def main():
    all_rows = []
    for filename in os.listdir(INPUT_DIR):
        if filename.endswith('.geojson'):
            filepath = os.path.join(INPUT_DIR, filename)
            rows = process_geojson_file(filepath)
            all_rows.extend(rows)
    # Write to CSV
    with open(OUTPUT_CSV, 'w', newline='', encoding='utf-8') as csvfile:
        fieldnames = ['forest', 'plot', 'pest', 'severity', 'date']
        writer = csv.DictWriter(csvfile, fieldnames=fieldnames)
        writer.writeheader()
        for row in all_rows:
            writer.writerow(row)
    print(f"Wrote {len(all_rows)} rows to {OUTPUT_CSV}")

if __name__ == '__main__':
    main()
