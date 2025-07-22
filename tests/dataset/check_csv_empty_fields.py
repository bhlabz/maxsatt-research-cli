import sys
import pandas as pd


def check_csv_for_empty_fields(csv_path):
    df = pd.read_csv(csv_path)
    empty_rows = df[df.isnull().any(axis=1) | (df == '').any(axis=1)]
    if empty_rows.empty:
        print("No rows with empty values found.")
        return
    for idx, row in empty_rows.iterrows():
        empty_cols = [col for col in df.columns if pd.isnull(row[col]) or row[col] == '']
        print(f"Row {idx+1} has empty values in columns: {empty_cols}")

if __name__ == "__main__":
    if len(sys.argv) != 2:
        print("Usage: python check_csv_empty_fields.py <csv_path>")
        sys.exit(1)
    csv_path = sys.argv[1]
    check_csv_for_empty_fields(csv_path) 