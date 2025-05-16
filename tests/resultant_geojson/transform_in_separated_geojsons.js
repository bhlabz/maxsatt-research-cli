const fs = require("fs");
const path = require("path");

const inputFolder = path.join(__dirname, "input");

function processGeoJSONFiles(folder) {
  fs.readdirSync(folder).forEach((fileOrFolder) => {
    const fullPath = path.join(folder, fileOrFolder);

    if (fs.statSync(fullPath).isDirectory()) {
      // If it's a directory, recursively process it
      processGeoJSONFiles(fullPath);
    } else if (path.extname(fileOrFolder).toLowerCase() === ".geojson") {
      // If it's a .geojson file, process it
      const data = JSON.parse(fs.readFileSync(fullPath, "utf8"));

      const pestFeaturesMap = {}; // { pestName: [features] }

      data.features.forEach((feature) => {
        const props = feature.properties;

        if (!props || !props.results) return;

        let parsedResults;
        try {
          parsedResults =
            typeof props.results === "string"
              ? JSON.parse(props.results)
              : props.results;
        } catch (err) {
          console.warn(
            `Invalid results format in file ${fileOrFolder}:`,
            err.message
          );
          return;
        }

        parsedResults.forEach(({ label, probability }) => {
          if (!label || probability == null) return;

          const pestName = label.trim();
          const clonedFeature = JSON.parse(JSON.stringify(feature));
          clonedFeature.properties = { label: pestName, probability };

          if (!pestFeaturesMap[pestName]) pestFeaturesMap[pestName] = [];
          pestFeaturesMap[pestName].push(clonedFeature);
        });
      });

      // Extract farm and plot from the file name
      const [farm1, farm2, farn3, plot] = fileOrFolder.split("_");
      const farm = `${farm1}_${farm2}_${farn3}`;
      // Write separate GeoJSON for each pest
      Object.entries(pestFeaturesMap).forEach(([pest, features]) => {
        const pestGeoJSON = {
          type: "FeatureCollection",
          features,
        };

        const pestFileName = `${pest
          .toLowerCase()
          .replace(/\s+/g, "_")}_${fileOrFolder}`;
        const outputPath = path.join(
          __dirname,
          "output",
          farm,
          plot,
          pestFileName
        );

        // Ensure the directory exists
        fs.mkdirSync(path.dirname(outputPath), { recursive: true });

        fs.writeFileSync(outputPath, JSON.stringify(pestGeoJSON, null, 2));
        console.log(`Saved ${features.length} features to ${outputPath}`);
      });
    }
  });
}

// Start processing from the input folder
processGeoJSONFiles(inputFolder);
