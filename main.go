package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type StockData struct {
	Date        string
	Open        string
	High        string
	Low         string
	Close       string
	Volume      string
	NewsTitle   string
	NewsSummary string
}

type AlphaVantageNewsResponse struct {
	Feed []struct {
		Title         string `json:"title"`
		URL           string `json:"url"`
		TimePublished string `json:"time_published"`
		Summary       string `json:"summary"`
	} `json:"feed"`
}

type AlphaVantageResponse struct {
	TimeSeries map[string]struct {
		Open   string `json:"1. open"`
		High   string `json:"2. high"`
		Low    string `json:"3. low"`
		Close  string `json:"4. close"`
		Volume string `json:"5. volume"`
	} `json:"Time Series (Daily)"`
}

type OpenFIGIRequest []struct {
	IdType  string `json:"idType"`
	IdValue string `json:"idValue"`
}

type OpenFIGIResponse []struct {
	Data []struct {
		Ticker string `json:"ticker"`
	} `json:"data"`
}

func main() {
	if err := godotenv.Load(); err != nil {
		fmt.Println("No .env file found. Will try using flags or environment variables.")
	}

	identifier := flag.String("id", "", "Stock identifier (WKN or ISIN)")
	days := flag.Int("days", 365, "Number of days to fetch data for")
	outputDir := flag.String("output-dir", ".", "Output directory for CSV files")
	apiKey := flag.String("apikey", os.Getenv("ALPHAVANTAGE_API_KEY"), "Alpha Vantage API key")
	flag.Parse()

	if *identifier == "" {
		fmt.Println("Please provide a stock identifier using -id flag")
		return
	}

	if *apiKey == "" {
		fmt.Println("Please provide an Alpha Vantage API key either:")
		fmt.Println("- in .env file as ALPHAVANTAGE_API_KEY=your_key")
		fmt.Println("- or using -apikey flag")
		return
	}

	symbol, err := getTickerSymbol(*identifier)
	if err != nil {
		fmt.Printf("Error looking up ticker symbol: %v\n", err)
		return
	}

	fmt.Printf("Found ticker symbol: %s\n", symbol)

	data, err := fetchStockData(symbol, *days, *apiKey)
	if err != nil {
		fmt.Printf("Error fetching stock data: %v\n", err)
		return
	}

	// Fetch and merge news data
	if err := enrichWithNews(symbol, *apiKey, data); err != nil {
		fmt.Printf("Warning: Error fetching news data: %v\n", err)
		// Continue anyway as we still have price data
	}

	filename := fmt.Sprintf("%s/%s.csv", *outputDir, symbol)
	if err := saveToCSV(data, filename); err != nil {
		fmt.Printf("Error saving to CSV: %v\n", err)
		return
	}

	fmt.Printf("Successfully saved stock data to %s\n", filename)
}

func enrichWithNews(symbol, apiKey string, data []StockData) error {
	url := fmt.Sprintf("https://www.alphavantage.co/query?function=NEWS_SENTIMENT&tickers=%s&apikey=%s",
		symbol, apiKey)

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("error making news request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading news response: %v", err)
	}

	var result AlphaVantageNewsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("error parsing news JSON: %v", err)
	}

	// Create a map of date to news items
	newsMap := make(map[string][]string)
	for _, item := range result.Feed {
		// Parse the timestamp (format: 20240308T130000)
		t, err := time.Parse("20060102T150405", item.TimePublished)
		if err != nil {
			continue
		}
		date := t.Format("2006-01-02")

		// Combine title and summary
		news := fmt.Sprintf("%s - %s", item.Title, item.Summary)
		newsMap[date] = append(newsMap[date], news)
	}

	// Merge news with stock data
	for i := range data {
		if news, exists := newsMap[data[i].Date]; exists && len(news) > 0 {
			// Take the first news item for the day
			parts := strings.SplitN(news[0], " - ", 2)
			if len(parts) == 2 {
				data[i].NewsTitle = parts[0]
				data[i].NewsSummary = parts[1]
			} else {
				data[i].NewsTitle = news[0]
			}
		}
	}

	return nil
}

func getTickerSymbol(identifier string) (string, error) {
	idType := "ID_WERTPAPIER"
	if len(identifier) == 12 && strings.HasPrefix(identifier, "US") {
		idType = "ID_ISIN"
	}

	requestBody := OpenFIGIRequest{
		{
			IdType:  idType,
			IdValue: identifier,
		},
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("error marshaling request: %v", err)
	}

	req, err := http.NewRequest("POST", "https://api.openfigi.com/v3/mapping", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	if apiKey := os.Getenv("OPENFIGI_API_KEY"); apiKey != "" {
		req.Header.Set("X-OPENFIGI-APIKEY", apiKey)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("OpenFIGI API returned status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response: %v", err)
	}

	var result OpenFIGIResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("error parsing JSON: %v", err)
	}

	if len(result) == 0 || len(result[0].Data) == 0 {
		return "", fmt.Errorf("no ticker found for identifier %s", identifier)
	}

	return result[0].Data[0].Ticker, nil
}

func fetchStockData(symbol string, days int, apiKey string) ([]StockData, error) {
	url := fmt.Sprintf("https://www.alphavantage.co/query?function=TIME_SERIES_DAILY&symbol=%s&outputsize=full&apikey=%s",
		symbol, apiKey)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %v", err)
	}

	var result AlphaVantageResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("error parsing JSON: %v", err)
	}

	if len(result.TimeSeries) == 0 {
		return nil, fmt.Errorf("no data returned for symbol %s", symbol)
	}

	var stockData []StockData
	cutoffDate := time.Now().AddDate(0, 0, -days)

	for date, data := range result.TimeSeries {
		parsedDate, err := time.Parse("2006-01-02", date)
		if err != nil {
			return nil, fmt.Errorf("error parsing date %s: %v", date, err)
		}

		if parsedDate.Before(cutoffDate) {
			continue
		}

		stockData = append(stockData, StockData{
			Date:   date,
			Open:   data.Open,
			High:   data.High,
			Low:    data.Low,
			Close:  data.Close,
			Volume: data.Volume,
		})
	}

	return stockData, nil
}

func saveToCSV(data []StockData, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	header := []string{"Date", "Open", "High", "Low", "Close", "Volume", "News Title", "News Summary"}
	if err := writer.Write(header); err != nil {
		return err
	}

	for _, record := range data {
		row := []string{
			record.Date,
			record.Open,
			record.High,
			record.Low,
			record.Close,
			record.Volume,
			record.NewsTitle,
			record.NewsSummary,
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}

	return nil
}
