package domain

import (
	"encoding/json"
	"math"
)

// Kopecks хранит денежную сумму в копейках. В JSON сериализуется как рубли (делится на 100).
type Kopecks int64

func (k Kopecks) MarshalJSON() ([]byte, error) {
	return json.Marshal(float64(k) / 100)
}

func (k *Kopecks) UnmarshalJSON(data []byte) error {
	var f float64
	if err := json.Unmarshal(data, &f); err != nil {
		return err
	}
	*k = Kopecks(math.Round(f * 100))
	return nil
}
