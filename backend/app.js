
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

app.get('/health', (req, res) => {
  res.status(200).json({ status: 'healthy' });
});

const PORT = process.env.PORT || 3000;

app.listen(PORT, '0.0.0.0', () => {
  console.log(`Server running on port ${PORT}`);
});
