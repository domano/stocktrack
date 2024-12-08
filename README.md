# Stock Data Tracker

A Go tool to fetch and track historical stock data and related news using WKN (Wertpapierkennummer) or ISIN identifiers.

## Features

- Fetch historical stock data using WKN or ISIN
- Automatic stock symbol lookup via OpenFIGI API
- Daily price data (Open, High, Low, Close, Volume)
- Related news matching trading days
- Configurable time range
- CSV output named after the stock symbol

## Prerequisites

- Go 1.16 or higher
- Alpha Vantage API key (get it [here](https://www.alphavantage.co/support/#api-key))
- Optional: OpenFIGI API key (get it [here](https://www.openfigi.com/api))

## Installation

```bash
# Clone the repository
git clone https://github.com/yourusername/stock-tracker
cd stock-tracker

# Install dependencies
go mod init stock-tracker
go get github.com/joho/godotenv
```

Create a `.env` file in the project root:
```env
ALPHAVANTAGE_API_KEY=your_key_here
OPENFIGI_API_KEY=your_key_here  # Optional
```

## Usage

Basic usage with default settings (365 days of data):
```bash
go run main.go -id US0378331005  # Using ISIN (creates AAPL.csv)
go run main.go -id 865985        # Using WKN (creates AAPL.csv)
```

Full options:
```bash
go run main.go -id <WKN/ISIN> [-days <number>] [-output-dir <path>] [-apikey <key>]
```

### Parameters

- `-id`: Stock identifier (WKN or ISIN) - required
- `-days`: Number of days of historical data (default: 365)
- `-output-dir`: Directory for CSV output (default: current directory)
- `-apikey`: Alpha Vantage API key (can also be set in .env file)

### Output Format

The tool creates a CSV file named `<SYMBOL>.csv` with the following columns:
- Date
- Open
- High
- Low
- Close
- Volume
- News Title
- News Summary

## API Keys

### Alpha Vantage
- Required for fetching stock data and news
- Free tier available
- Get your key at: https://www.alphavantage.co/support/#api-key

### OpenFIGI
- Optional but recommended for better WKN/ISIN lookup
- Free tier available
- Get your key at: https://www.openfigi.com/api

## Rate Limits

- Alpha Vantage free tier: 5 API calls per minute, 500 per day
- OpenFIGI without API key: 5 requests per minute
- OpenFIGI with API key: 25 requests per second

## Example

```bash
# Fetch 180 days of Apple stock data using ISIN
go run main.go -id US0378331005 -days 180 -output-dir ./data

# This will create data/AAPL.csv with daily prices and relevant news
```

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details.
