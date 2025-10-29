
const fs = require("fs");

// add price: 0 to info.json files

function addPrices() {
  const appsDir = "./apps";
  const appNames = fs.readdirSync(appsDir);

  appNames.forEach((appName) => {
    const infoPath = `${appsDir}/${appName}/info.json`;
    if (fs.existsSync(infoPath)) {
      const infoData = JSON.parse(fs.readFileSync(infoPath, "utf-8"));
      if ("price" in infoData) {
        console.log(`Price already exists for app: ${appName}`);
        return;
      }
      infoData.price = 0;
      fs.writeFileSync(infoPath, JSON.stringify(infoData, null, 2), "utf-8");
      console.log(`Added price to info.json for app: ${appName}`);
    }
  });
}

addPrices();