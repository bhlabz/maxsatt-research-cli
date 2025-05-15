const fs = require("fs");
const path = require("path");

const inputFolder = path.join(__dirname, "input");

fs.readdirSync(inputFolder).forEach((file) => {
  if (path.extname(file).toLowerCase() !== ".geojson") return;

  const inputPath = path.join(inputFolder, file);
  const data = JSON.parse(fs.readFileSync(inputPath, "utf8"));

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
      console.warn(`Invalid results format in file ${file}:`, err.message);
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

  // Write separate GeoJSON for each pest
  Object.entries(pestFeaturesMap).forEach(([pest, features]) => {
    const pestGeoJSON = {
      type: "FeatureCollection",
      features,
    };

    const fileName = `output/${pest
      .toLowerCase()
      .replace(/\s+/g, "_")}_${file}`;
    const outputPath = path.join(__dirname, fileName);

    fs.writeFileSync(outputPath, JSON.stringify(pestGeoJSON, null, 2));
    console.log(`Saved ${features.length} features to ${fileName}`);
  });
});
