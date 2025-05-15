const fs = require("fs");
const path = require("path");

const inputFile =
  "Fazendas_Manulife_030_2025-05-01_166_2025-05-12_0_20.geojson";

const inputPath = path.join(__dirname, "input/" + inputFile);
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
    console.warn("Invalid results format:", err.message);
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
    .replace(/\s+/g, "_")}_${inputFile}`;
  const outputPath = path.join(__dirname, fileName);

  fs.writeFileSync(outputPath, JSON.stringify(pestGeoJSON, null, 2));
  console.log(`Saved ${features.length} features to ${fileName}`);
});
