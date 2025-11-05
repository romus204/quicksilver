// Пакет calculate содержит логику решения задачи маршрутизации курьеров (VPR).
// Алгоритм жадно назначает задачи первому подходящему курьеру с учётом:
// - вместимости,
// - временных окон (сборка и доставка),
// - времени движения между точками.
package greedy

import (
	"math"
	"time"
)

const (
	// avgSpeedMps — средняя скорость курьера в м/с (~15 км/ч)
	avgSpeedMps = 4.17
	// earthRadius — радиус Земли в метрах
	earthRadius = 6371000.0
)

// Slot представляет временной интервал (например, окно доставки или сборки)
type Slot struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
}

// Coordinates — географические координаты точки
type Coordinates struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

// Capacity — объём и вес груза
type Capacity struct {
	Volume float64 `json:"volume"`
	Weight float64 `json:"weight"`
}

// Courier — данные о курьере
type Courier struct {
	Guid           string       `json:"guid"`
	StartPoint     *Coordinates `json:"start_point"`
	FinishPoint    *Coordinates `json:"finish_point,omitempty"`
	Priority       *int         `json:"priority,omitempty"`
	Capacity       *Capacity    `json:"capacity,omitempty"`
	PickupDuration int64        `json:"pickup_duration"` // время на сбор (сек)
	DropDuration   int64        `json:"drop_duration"`   // время на доставку (сек)
}

// Task — задание по доставке от отправителя к получателю
type Task struct {
	Guid           string       `json:"guid"`
	SenderPoint    *Coordinates `json:"sender_point"`
	RecipientPoint *Coordinates `json:"recipient_point"`
	Capacity       *Capacity    `json:"capacity,omitempty"`
	Assembly       *Slot        `json:"assembly"` // окно, когда можно забрать
	Slot           *Slot        `json:"slot"`     // окно доставки
	PickupDuration *int64       `json:"pickup_duration,omitempty"`
	DropDuration   *int64       `json:"drop_duration,omitempty"`
}

// Request — входные данные для решателя
type Request struct {
	Couriers []*Courier `json:"couriers"`
	Tasks    []*Task    `json:"tasks"`
}

// Route — маршрут курьера (упорядоченный список задач)
type Route struct {
	CourierGuid string   `json:"courier_guid"`
	Route       []string `json:"route"` // task_guid в порядке выполнения
}

// Response — результат решения
type Response struct {
	Routes     []Route  `json:"routes"`
	Unassigned []string `json:"unassigned"` // GUID задач, не назначенных ни одному курьеру
}

// DistanceTo вычисляет расстояние между двумя точками на поверхности Земли (в метрах)
func (c *Coordinates) DistanceTo(other *Coordinates) float64 {
	if c == nil || other == nil {
		return 0
	}
	lat1 := c.Lat * math.Pi / 180
	lon1 := c.Lon * math.Pi / 180
	lat2 := other.Lat * math.Pi / 180
	lon2 := other.Lon * math.Pi / 180

	dlat := lat2 - lat1
	dlon := lon2 - lon1

	a := math.Sin(dlat/2)*math.Sin(dlat/2) +
		math.Cos(lat1)*math.Cos(lat2)*math.Sin(dlon/2)*math.Sin(dlon/2)
	cVal := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadius * cVal
}

// Add суммирует две ёмкости
func (cap *Capacity) Add(other *Capacity) *Capacity {
	if cap == nil {
		return other
	}
	if other == nil {
		return cap
	}
	return &Capacity{
		Volume: cap.Volume + other.Volume,
		Weight: cap.Weight + other.Weight,
	}
}

// CanFit проверяет, помещается ли другой объём в текущий
func (cap *Capacity) CanFit(other *Capacity) bool {
	if other == nil {
		return true
	}
	if cap == nil {
		return other.Volume == 0 && other.Weight == 0
	}
	return cap.Volume >= other.Volume && cap.Weight >= other.Weight
}

// SolveVPR решает задачу маршрутизации методом жадного назначения.
// Каждая задача назначается первому курьеру, который может её выполнить.
func SolveVPR(req Request) *Response {
	var unassigned []string

	// Внутреннее состояние каждого курьера
	type courierState struct {
		*Courier
		currentLocation *Coordinates // текущее местоположение
		currentTimeSec  int64        // текущее время в секундах от старта
		usedCapacity    *Capacity    // занятый объём
		taskGuids       []string     // список назначенных задач
	}

	// Инициализация состояния всех курьеров
	states := make([]*courierState, len(req.Couriers))
	for i, courier := range req.Couriers {
		states[i] = &courierState{
			Courier:         courier,
			currentLocation: courier.StartPoint,
			currentTimeSec:  0,
			usedCapacity:    &Capacity{},
			taskGuids:       nil,
		}
	}

	// Перебираем все задачи
	for _, task := range req.Tasks {
		// Определяем длительность операций (берём из задачи или используем дефолт из первого курьера)
		pickupDur := getOrDefault(task.PickupDuration, req.Couriers[0].PickupDuration)
		dropDur := getOrDefault(task.DropDuration, req.Couriers[0].DropDuration)

		// Преобразуем временные окна в Unix-время (секунды)
		var assemblyFrom, assemblyTo, slotFrom, slotTo *int64
		if task.Assembly != nil {
			f, t := task.Assembly.From.Unix(), task.Assembly.To.Unix()
			assemblyFrom, assemblyTo = &f, &t
		}
		if task.Slot != nil {
			f, t := task.Slot.From.Unix(), task.Slot.To.Unix()
			slotFrom, slotTo = &f, &t
		}

		assigned := false
		// Пытаемся назначить задачу каждому курьеру
		for _, state := range states {
			// Проверка ёмкости
			newCap := state.usedCapacity.Add(task.Capacity)
			if state.Capacity != nil && !state.Capacity.CanFit(newCap) {
				continue
			}

			// Время движения до отправителя
			travelToSender := int64(state.currentLocation.DistanceTo(task.SenderPoint) / avgSpeedMps)
			arrivalSender := state.currentTimeSec + travelToSender

			// Начало работы — как только прибыл или после начала окна сборки
			startWork := arrivalSender
			if assemblyFrom != nil && arrivalSender < *assemblyFrom {
				startWork = *assemblyFrom
			}
			// Если начало работы после окончания окна — пропускаем
			if assemblyTo != nil && startWork > *assemblyTo {
				continue
			}

			// Время ухода со сборки
			departSender := startWork + pickupDur

			// Время движения до получателя
			travelToRecipient := int64(task.SenderPoint.DistanceTo(task.RecipientPoint) / avgSpeedMps)
			arrivalRecipient := departSender + travelToRecipient

			// Время доставки — соблюдение окна доставки
			deliverTime := arrivalRecipient
			if slotFrom != nil && arrivalRecipient < *slotFrom {
				deliverTime = *slotFrom
			}
			if slotTo != nil && deliverTime > *slotTo {
				continue
			}

			// Общее время завершения задачи
			finishTime := deliverTime + dropDur

			// Обновляем состояние курьера
			state.currentLocation = task.RecipientPoint
			state.currentTimeSec = finishTime
			state.usedCapacity = newCap
			state.taskGuids = append(state.taskGuids, task.Guid)

			assigned = true
			break // Задача назначена, переходим к следующей
		}

		if !assigned {
			unassigned = append(unassigned, task.Guid)
		}
	}

	// Формируем выходной маршрут
	var routes []Route
	for _, state := range states {
		if len(state.taskGuids) > 0 {
			routes = append(routes, Route{
				CourierGuid: state.Guid,
				Route:       state.taskGuids,
			})
		}
	}

	return &Response{
		Routes:     routes,
		Unassigned: unassigned,
	}
}

// getOrDefault возвращает значение или дефолт, если указатель nil
func getOrDefault(ptr *int64, def int64) int64 {
	if ptr != nil {
		return *ptr
	}
	return def
}
