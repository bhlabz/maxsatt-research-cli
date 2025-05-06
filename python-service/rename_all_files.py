import os

for filename in os.listdir():
    if filename.endswith(".csv.csv"):
        new_name = filename.replace(".csv.csv", "_a_b.csv")
        os.rename(filename, new_name)