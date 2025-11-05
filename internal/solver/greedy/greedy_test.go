package greedy

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCoordinates_DistanceTo(t *testing.T) {
	moscow := &Coordinates{Lat: 55.7558, Lon: 37.6176}
	spb := &Coordinates{Lat: 59.9343, Lon: 30.3351}

	distance := moscow.DistanceTo(spb)
	assert.Greater(t, distance, 600_000.0) // > 600 км
	assert.Less(t, distance, 800_000.0)    // < 800 км

	// Тест на одинаковые точки
	same := moscow.DistanceTo(moscow)
	assert.Equal(t, 0.0, same)
}

func TestCapacity_Add(t *testing.T) {
	c1 := &Capacity{Volume: 2.0, Weight: 3.0}
	c2 := &Capacity{Volume: 1.5, Weight: 0.5}

	result := c1.Add(c2)
	assert.Equal(t, 3.5, result.Volume)
	assert.Equal(t, 3.5, result.Weight)

	// nil + объект
	assert.Equal(t, c1, (*Capacity)(nil).Add(c1))
	assert.Equal(t, c2, c2.Add(nil))

	// nil + nil
	assert.Nil(t, (*Capacity)(nil).Add(nil))
}

func TestCapacity_CanFit(t *testing.T) {
	container := &Capacity{Volume: 10.0, Weight: 20.0}

	// Влезает
	item1 := &Capacity{Volume: 5.0, Weight: 10.0}
	assert.True(t, container.CanFit(item1))

	// Не влезает по объёму
	item2 := &Capacity{Volume: 15.0, Weight: 5.0}
	assert.False(t, container.CanFit(item2))

	// Не влезает по весу
	item3 := &Capacity{Volume: 5.0, Weight: 25.0}
	assert.False(t, container.CanFit(item3))

	// nil всегда помещается
	assert.True(t, container.CanFit(nil))
	assert.True(t, (*Capacity)(nil).CanFit(&Capacity{Volume: 0, Weight: 0}))
	assert.False(t, (*Capacity)(nil).CanFit(&Capacity{Volume: 1, Weight: 0}))
}

func TestSolveVPR_BasicAssignment(t *testing.T) {
	now := time.Now()

	courier := &Courier{
		Guid:           "c1",
		StartPoint:     &Coordinates{Lat: 55.75, Lon: 37.62},
		Capacity:       &Capacity{Volume: 10, Weight: 10},
		PickupDuration: 120,
		DropDuration:   60,
	}

	task := &Task{
		Guid:           "t1",
		SenderPoint:    &Coordinates{Lat: 55.75, Lon: 37.63}, // рядом
		RecipientPoint: &Coordinates{Lat: 55.76, Lon: 37.64},
		Capacity:       &Capacity{Volume: 2, Weight: 3},
		Assembly: &Slot{
			From: now,
			To:   now.Add(2 * time.Hour),
		},
		Slot: &Slot{
			From: now.Add(30 * time.Minute),
			To:   now.Add(3 * time.Hour),
		},
	}

	req := Request{
		Couriers: []*Courier{courier},
		Tasks:    []*Task{task},
	}

	resp := SolveVPR(req)

	assert.Len(t, resp.Routes, 1)
	assert.Equal(t, "c1", resp.Routes[0].CourierGuid)
	assert.Contains(t, resp.Routes[0].Route, "t1")
	assert.Empty(t, resp.Unassigned)
}

func TestSolveVPR_UnassignedDueToCapacity(t *testing.T) {
	courier := &Courier{
		Guid:           "c1",
		StartPoint:     &Coordinates{Lat: 55.75, Lon: 37.62},
		Capacity:       &Capacity{Volume: 5, Weight: 5},
		PickupDuration: 120,
		DropDuration:   60,
	}

	task := &Task{
		Guid:           "t1",
		SenderPoint:    &Coordinates{Lat: 55.75, Lon: 37.63},
		RecipientPoint: &Coordinates{Lat: 55.76, Lon: 37.64},
		Capacity:       &Capacity{Volume: 10, Weight: 10}, // больше, чем может курьер
	}

	req := Request{
		Couriers: []*Courier{courier},
		Tasks:    []*Task{task},
	}

	resp := SolveVPR(req)

	assert.Empty(t, resp.Routes)
	assert.Contains(t, resp.Unassigned, "t1")
}

func TestSolveVPR_UnassignedDueToTimeWindow(t *testing.T) {
	now := time.Now()

	courier := &Courier{
		Guid:           "c1",
		StartPoint:     &Coordinates{Lat: 55.75, Lon: 37.62},
		Capacity:       &Capacity{Volume: 10, Weight: 10},
		PickupDuration: 120,
		DropDuration:   60,
	}

	// Очень узкое окно доставки — прибытие будет позже
	task := &Task{
		Guid:           "t1",
		SenderPoint:    &Coordinates{Lat: 55.75, Lon: 37.63},
		RecipientPoint: &Coordinates{Lat: 55.76, Lon: 37.64},
		Assembly: &Slot{
			From: now,
			To:   now.Add(10 * time.Second), // очень короткое окно сборки
		},
		Slot: &Slot{
			From: now,
			To:   now.Add(1 * time.Minute), // слишком рано закончится
		},
	}

	req := Request{
		Couriers: []*Courier{courier},
		Tasks:    []*Task{task},
	}

	resp := SolveVPR(req)

	assert.Empty(t, resp.Routes)
	assert.Contains(t, resp.Unassigned, "t1")
}

func TestSolveVPR_MultipleCouriers_PicksFirstAvailable(t *testing.T) {
	now := time.Now()

	courier1 := &Courier{
		Guid:           "c1",
		StartPoint:     &Coordinates{Lat: 55.75, Lon: 37.62},
		Capacity:       &Capacity{Volume: 1, Weight: 1}, // маленькая ёмкость
		PickupDuration: 120,
		DropDuration:   60,
	}

	courier2 := &Courier{
		Guid:           "c2",
		StartPoint:     &Coordinates{Lat: 55.75, Lon: 37.62},
		Capacity:       &Capacity{Volume: 10, Weight: 10}, // подходит
		PickupDuration: 120,
		DropDuration:   60,
	}

	task := &Task{
		Guid:           "t1",
		SenderPoint:    &Coordinates{Lat: 55.75, Lon: 37.63},
		RecipientPoint: &Coordinates{Lat: 55.76, Lon: 37.64},
		Capacity:       &Capacity{Volume: 5, Weight: 5},
		Assembly: &Slot{
			From: now,
			To:   now.Add(2 * time.Hour),
		},
		Slot: &Slot{
			From: now,
			To:   now.Add(3 * time.Hour),
		},
	}

	req := Request{
		Couriers: []*Courier{courier1, courier2},
		Tasks:    []*Task{task},
	}

	resp := SolveVPR(req)

	assert.Len(t, resp.Routes, 1)
	assert.Equal(t, "c2", resp.Routes[0].CourierGuid) // второй курьер — больше ёмкость
	assert.Contains(t, resp.Routes[0].Route, "t1")
}
