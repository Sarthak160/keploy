const fs = require('fs');

const coverageJson = fs.readFileSync('/Users/sarthak_1/Documents/Keploy/trash/samples-typescript/node-fetch/.nyc_output/aa1a9f43-7602-48a6-84bd-6b7c65befdd2.json', 'utf8');
const coverage = JSON.parse(coverageJson);

const executedLinesByFile = {};

for (const filePath of Object.keys(coverage)) {
  const executedLines = new Set();
  const fileCoverage = coverage[filePath];
  const statementMap = fileCoverage.statementMap;
  const hitCounts = fileCoverage.s;

  for (const statementId in statementMap) {
    if (hitCounts[statementId] > 0) {
      const executedLine = statementMap[statementId].start.line;
      executedLines.add(executedLine);
    }
  }

  executedLinesByFile[filePath] = Array.from(executedLines).sort((a, b) => a - b);
}

console.log("Executed lines by file:", executedLinesByFile);
