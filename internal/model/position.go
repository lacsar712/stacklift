package model

import "math"

type Position struct {
	TravelMM int64
	HoistMM  int64
	ForkMM   int64
}

func LocationToPosition(loc Location, bayPitch, levelPitch, depthPitch int64) Position {
	return Position{
		TravelMM: int64(loc.Bay) * bayPitch,
		HoistMM:  int64(loc.Level) * levelPitch,
		ForkMM:   int64(loc.Depth) * depthPitch,
	}
}

func PositionToLocation(pos Position, aisle AisleID, bayPitch, levelPitch, depthPitch int64) Location {
	bay := int(pos.TravelMM / bayPitch)
	if bay < 1 {
		bay = 1
	}
	level := int(pos.HoistMM / levelPitch)
	if level < 1 {
		level = 1
	}
	depth := int(pos.ForkMM / depthPitch)
	if depth < 0 {
		depth = 0
	}
	return Location{Aisle: aisle, Bay: bay, Level: level, Depth: depth}
}

func DistanceMM(a, b Position) int64 {
	dt := a.TravelMM - b.TravelMM
	dh := a.HoistMM - b.HoistMM
	df := a.ForkMM - b.ForkMM
	return int64(math.Sqrt(float64(dt*dt + dh*dh + df*df)))
}

func TravelDelta(from, to Position) int64 {
	d := to.TravelMM - from.TravelMM
	if d < 0 {
		return -d
	}
	return d
}

func HoistDelta(from, to Position) int64 {
	d := to.HoistMM - from.HoistMM
	if d < 0 {
		return -d
	}
	return d
}

func ForkDelta(from, to Position) int64 {
	d := to.ForkMM - from.ForkMM
	if d < 0 {
		return -d
	}
	return d
}

func ClampMM(value, min, max int64) int64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func PoseToPosition(pose CranePose) Position {
	return Position{TravelMM: pose.TravelMM, HoistMM: pose.HoistMM, ForkMM: pose.ForkMM}
}

func ApplyPosition(pose CranePose, pos Position) CranePose {
	pose.TravelMM = pos.TravelMM
	pose.HoistMM = pos.HoistMM
	pose.ForkMM = pos.ForkMM
	return pose
}
