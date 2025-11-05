# Quicksilver — Courier Routing Optimization Service
A lightweight Go service for solving the Vehicle Routing Problem (VRP) — optimal assignment of delivery tasks to couriers, considering geolocation, time windows, and capacity constraints.

## Available Algorithms
- Greedy Assignment Algorithm

## Features
- Accepts a list of couriers and tasks via HTTP API.
- Assigns tasks to couriers using a greedy algorithm.
- Takes into account:
 - Distance between points (Haversine formula).
 - Time windows (pickup and delivery).
 - Courier capacity (volume and weight).
 - Task durations (pickup and drop-off times).

 ## API

 `POST /vpr/greedy/`

 requset:

 ```json
 {
  "couriers": [
    {
      "guid": "c1",
      "start_point": { "lat": 55.75, "lon": 37.62 },
      "capacity": { "volume": 10, "weight": 20 },
      "pickup_duration": 120,
      "drop_duration": 60
    }
  ],
  "tasks": [
    {
      "guid": "t1",
      "sender_point": { "lat": 55.76, "lon": 37.60 },
      "recipient_point": { "lat": 55.78, "lon": 37.65 },
      "capacity": { "volume": 2, "weight": 3 },
      "assembly": {
        "from": "2025-04-05T10:00:00Z",
        "to": "2025-04-05T12:00:00Z"
      },
      "slot": {
        "from": "2025-04-05T11:00:00Z",
        "to": "2025-04-05T14:00:00Z"
      }
    }
  ]
}
 ```

 response: 

 ```json

{
  "routes": [
    {
      "courier_guid": "c1",
      "route": [
        "t1"
      ]
    }
  ],
  "unassigned": null
}

 ```

 ## TODO
 - More advanced distribution algorithms
 - More options for fine-tuning
