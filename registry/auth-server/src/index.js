const express = require("express");
const app = express();
const port = 8000;

app.get("/", (req, res) => {
  console.log("my headers:", req.headers);

  if (!req.headers.authorization) {
    return res.status(401).send();
  }

  const b64auth = (req.headers.authorization || "").split(" ")[1] || "";
  const [login, password] = Buffer.from(b64auth, "base64")
    .toString()
    .split(":");
  console.log(`login: ${login} password: ${password}`);
  res.status(200).send();
});

app.listen(port, () => {
  console.log(`Example app listening on port ${port}`);
});
