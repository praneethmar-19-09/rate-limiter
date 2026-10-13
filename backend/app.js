const express = require('express');
const app = express();

app.get('/users', (req, res) => {
  res.send('Praneeth');
});

app.get('/products', (req, res) => {
  res.send('Products endpoint');
});

app.get('/', (req, res) => {
  res.send('Hello, world!');
});

app.listen(3000, () => {
  console.log('Server running on port 3000');
});