package calculator

import (
	"math"
	"time"

	"github.com/shopspring/decimal"

	"10.0.0.50/tuan.quang.tran/vms-v2/internal/models"
)

func CalculateStorageBytes(credit decimal.Decimal, hours float64) int64 {
	oneUSD := decimal.New(1, 18)
	creditUSD := credit.Div(oneUSD)

	storagePerBytePerHour := decimal.NewFromInt(models.HubCostPerStorage).
		Div(decimal.New(1024*1024*1024*1024, 0))
	hoursDec := decimal.NewFromFloat(hours)
	costPerByteForGivenHours := storagePerBytePerHour.Mul(hoursDec)
	storageBytes := creditUSD.Div(costPerByteForGivenHours)

	storageInt64, _ := storageBytes.Floor().Float64()
	return int64(math.Floor(storageInt64))
}

func CalculateDeliveryBytes(credit decimal.Decimal) int64 {
	oneUSD := decimal.New(1, 18)
	creditUSD := credit.Div(oneUSD)

	deliveryPerByte := decimal.NewFromInt(models.HubCostPerDelivery).
		Div(decimal.New(1024*1024*1024*1024, 0))
	deliveryBytes := creditUSD.Div(deliveryPerByte)

	deliveryInt64, _ := deliveryBytes.Floor().Float64()
	return int64(math.Floor(deliveryInt64))
}

func ConvertPriceToUSD(price decimal.Decimal) float64 {
	oneUSD := decimal.New(1, 18)
	priceUSD := price.Div(oneUSD)

	result, _ := priceUSD.Float64()
	return result
}

func CalculateExpiredTimeToEndOfMonth() time.Duration {
	now := time.Now()
	currentYear, currentMonth, _ := now.Date()
	currentLocation := now.Location()

	firstOfNextMonth := time.Date(currentYear, currentMonth+1, 1, 0, 0, 0, 0, currentLocation)

	return firstOfNextMonth.Sub(now)
}
