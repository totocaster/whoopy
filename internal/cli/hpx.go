package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"strings"
	"time"

	"github.com/toto/whoopy/internal/cycles"
	"github.com/toto/whoopy/internal/profile"
	"github.com/toto/whoopy/internal/recovery"
	"github.com/toto/whoopy/internal/sleep"
	"github.com/toto/whoopy/internal/workouts"
)

const (
	hpxSource = "whoop"
	kjToKcal  = 0.239005736
)

type hpxWriter struct {
	enc                *json.Encoder
	seenSignpostRecord map[string]struct{}
}

type hpxMetricRecord struct {
	Table      string         `json:"table"`
	Date       string         `json:"date"`
	Key        string         `json:"key"`
	Value      float64        `json:"value"`
	Source     string         `json:"source"`
	OriginID   string         `json:"origin_id,omitempty"`
	TS         string         `json:"ts"`
	SignpostID *string        `json:"signpost_id"`
	Meta       map[string]any `json:"meta"`
}

type hpxSignpostRecord struct {
	Table    string         `json:"table"`
	Kind     string         `json:"kind"`
	TS       string         `json:"ts"`
	Edge     string         `json:"edge"`
	Source   string         `json:"source"`
	OriginID string         `json:"origin_id,omitempty"`
	ID       string         `json:"id,omitempty"`
	Data     map[string]any `json:"data"`
}

func newHPXWriter(w io.Writer) *hpxWriter {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return &hpxWriter{
		enc:                enc,
		seenSignpostRecord: make(map[string]struct{}),
	}
}

func (w *hpxWriter) write(record any) error {
	return w.enc.Encode(record)
}

func (w *hpxWriter) writeSignpost(record hpxSignpostRecord) error {
	key := strings.TrimSpace(record.OriginID)
	if key == "" {
		key = fmt.Sprintf("%s|%s|%s|%s|%s", record.Kind, record.Edge, record.TS, record.Source, record.ID)
	}
	if _, exists := w.seenSignpostRecord[key]; exists {
		return nil
	}
	w.seenSignpostRecord[key] = struct{}{}
	return w.write(record)
}

func (w *hpxWriter) writeSleepSession(sess sleep.Session) error {
	startID := fmt.Sprintf("sleep-%s-start", sess.ID)
	signpostID := stringPtr(startID)
	metricTS := sleepMetricTimestamp(sess)
	data := compactMap(map[string]any{
		"is_nap":                sess.Nap,
		"timezone_offset":       trimToNil(sess.TimezoneOffset),
		"cycle_id":              nonZeroInt64(sess.CycleID),
		"score_state":           trimToNil(sess.ScoreState),
		"sleep_performance_pct": floatPtrValue(sess.Score.SleepPerformancePercentage),
		"sleep_consistency_pct": floatPtrValue(sess.Score.SleepConsistencyPercentage),
		"respiratory_rate_rpm":  floatPtrValue(sess.Score.RespiratoryRate),
		"sleep_cycle_count":     intPtrValue(sess.Score.StageSummary.SleepCycleCount),
		"disturbance_count":     intPtrValue(sess.Score.StageSummary.DisturbanceCount),
		"total_no_data_ms":      int64PtrValue(sess.Score.StageSummary.TotalNoDataTimeMilli),
	})
	if !sess.Start.IsZero() {
		if err := w.writeSignpost(hpxSignpostRecord{
			Table:    "signpost",
			Kind:     "sleep",
			TS:       formatHPXTimestamp(sess.Start),
			Edge:     "start",
			Source:   hpxSource,
			OriginID: fmt.Sprintf("whoop-sleep-%s-start", sess.ID),
			ID:       startID,
			Data:     data,
		}); err != nil {
			return err
		}
	} else {
		signpostID = nil
	}
	if !sess.End.IsZero() {
		if err := w.writeSignpost(hpxSignpostRecord{
			Table:    "signpost",
			Kind:     "sleep",
			TS:       formatHPXTimestamp(sess.End),
			Edge:     "end",
			Source:   hpxSource,
			OriginID: fmt.Sprintf("whoop-sleep-%s-end", sess.ID),
			ID:       fmt.Sprintf("sleep-%s-end", sess.ID),
			Data:     data,
		}); err != nil {
			return err
		}
	}

	date := sleepMetricDate(sess)
	if date == "" {
		return nil
	}
	if value, ok := durationMillis(sess.Start, sess.End); ok {
		if err := w.writeMetric(date, "sleep.duration_ms", value, metricTS, fmt.Sprintf("whoop-sleep-%s-duration", sess.ID), signpostID, nil); err != nil {
			return err
		}
	}
	if value := int64PtrValue(sess.Score.StageSummary.TotalInBedTimeMilli); value != nil {
		if err := w.writeMetric(date, "sleep.time_in_bed_ms", *value, metricTS, fmt.Sprintf("whoop-sleep-%s-time-in-bed", sess.ID), signpostID, nil); err != nil {
			return err
		}
	}
	if value := floatPtrValue(sess.Score.SleepEfficiencyPercentage); value != nil {
		if err := w.writeMetric(date, "sleep.efficiency_pct", *value, metricTS, fmt.Sprintf("whoop-sleep-%s-efficiency", sess.ID), signpostID, nil); err != nil {
			return err
		}
	}
	if value := int64PtrValue(sess.Score.StageSummary.TotalRemSleepTimeMilli); value != nil {
		if err := w.writeMetric(date, "sleep.rem_ms", *value, metricTS, fmt.Sprintf("whoop-sleep-%s-rem", sess.ID), signpostID, nil); err != nil {
			return err
		}
	}
	if value := int64PtrValue(sess.Score.StageSummary.TotalLightSleepTimeMilli); value != nil {
		if err := w.writeMetric(date, "sleep.light_ms", *value, metricTS, fmt.Sprintf("whoop-sleep-%s-light", sess.ID), signpostID, nil); err != nil {
			return err
		}
	}
	if value := int64PtrValue(sess.Score.StageSummary.TotalSlowWaveSleepTimeMilli); value != nil {
		if err := w.writeMetric(date, "sleep.deep_ms", *value, metricTS, fmt.Sprintf("whoop-sleep-%s-deep", sess.ID), signpostID, nil); err != nil {
			return err
		}
	}
	if value := int64PtrValue(sess.Score.StageSummary.TotalAwakeTimeMilli); value != nil {
		if err := w.writeMetric(date, "sleep.awake_ms", *value, metricTS, fmt.Sprintf("whoop-sleep-%s-awake", sess.ID), signpostID, nil); err != nil {
			return err
		}
	}
	return nil
}

func (w *hpxWriter) writeWorkout(workout workouts.Workout) error {
	startID := fmt.Sprintf("workout-%s-start", workout.ID)
	signpostID := stringPtr(startID)
	metricTS := workoutMetricTimestamp(workout)
	data := compactMap(map[string]any{
		"sport":                 trimToNil(workout.SportName),
		"sport_id":              intPtrValue(workout.SportID),
		"timezone_offset":       trimToNil(workout.TimezoneOffset),
		"score_state":           trimToNil(workout.ScoreState),
		"v1_id":                 int64PtrValue(workout.V1ID),
		"percent_recorded":      floatPtrValue(workout.Score.PercentRecorded),
		"distance_meter":        floatPtrValue(workout.Score.DistanceMeter),
		"altitude_gain_meter":   floatPtrValue(workout.Score.AltitudeGainMeter),
		"altitude_change_meter": floatPtrValue(workout.Score.AltitudeChangeMeter),
		"zone_zero_ms":          int64PtrValue(workout.Score.ZoneDurations.ZoneZeroMilli),
		"zone_one_ms":           int64PtrValue(workout.Score.ZoneDurations.ZoneOneMilli),
		"zone_two_ms":           int64PtrValue(workout.Score.ZoneDurations.ZoneTwoMilli),
		"zone_three_ms":         int64PtrValue(workout.Score.ZoneDurations.ZoneThreeMilli),
		"zone_four_ms":          int64PtrValue(workout.Score.ZoneDurations.ZoneFourMilli),
		"zone_five_ms":          int64PtrValue(workout.Score.ZoneDurations.ZoneFiveMilli),
	})
	if !workout.Start.IsZero() {
		if err := w.writeSignpost(hpxSignpostRecord{
			Table:    "signpost",
			Kind:     "workout",
			TS:       formatHPXTimestamp(workout.Start),
			Edge:     "start",
			Source:   hpxSource,
			OriginID: fmt.Sprintf("whoop-workout-%s-start", workout.ID),
			ID:       startID,
			Data:     data,
		}); err != nil {
			return err
		}
	} else {
		signpostID = nil
	}
	if !workout.End.IsZero() {
		if err := w.writeSignpost(hpxSignpostRecord{
			Table:    "signpost",
			Kind:     "workout",
			TS:       formatHPXTimestamp(workout.End),
			Edge:     "end",
			Source:   hpxSource,
			OriginID: fmt.Sprintf("whoop-workout-%s-end", workout.ID),
			ID:       fmt.Sprintf("workout-%s-end", workout.ID),
			Data:     data,
		}); err != nil {
			return err
		}
	}

	date := metricDateFromTimestamp(workout.Start, workout.End)
	if date == "" {
		return nil
	}
	if value, ok := durationMillis(workout.Start, workout.End); ok {
		if err := w.writeMetric(date, "workout.duration_ms", value, metricTS, fmt.Sprintf("whoop-workout-%s-duration", workout.ID), signpostID, nil); err != nil {
			return err
		}
	}
	if workout.Score.Strain > 0 {
		if err := w.writeMetric(date, "workout.strain_score", workout.Score.Strain, metricTS, fmt.Sprintf("whoop-workout-%s-strain", workout.ID), signpostID, nil); err != nil {
			return err
		}
	}
	if value := intPtrValue(workout.Score.AverageHeartRate); value != nil {
		if err := w.writeMetric(date, "workout.avg_hr_bpm", *value, metricTS, fmt.Sprintf("whoop-workout-%s-avg-hr", workout.ID), signpostID, nil); err != nil {
			return err
		}
	}
	if value := intPtrValue(workout.Score.MaxHeartRate); value != nil {
		if err := w.writeMetric(date, "workout.max_hr_bpm", *value, metricTS, fmt.Sprintf("whoop-workout-%s-max-hr", workout.ID), signpostID, nil); err != nil {
			return err
		}
	}
	if value := floatPtrValue(workout.Score.Kilojoule); value != nil {
		if err := w.writeMetric(date, "workout.kilojoules", *value, metricTS, fmt.Sprintf("whoop-workout-%s-kilojoules", workout.ID), signpostID, nil); err != nil {
			return err
		}
		if err := w.writeMetric(date, "workout.calories_kcal", *value*kjToKcal, metricTS, fmt.Sprintf("whoop-workout-%s-calories", workout.ID), signpostID, map[string]any{
			"derived_from": "kilojoules",
		}); err != nil {
			return err
		}
	}
	return nil
}

func (w *hpxWriter) writeRecovery(rec recovery.Recovery, cycle *cycles.Cycle) error {
	signpostID := (*string)(nil)
	metricTS := recoveryMetricTimestamp(rec, cycle)
	if cycle != nil {
		startID := fmt.Sprintf("recovery-window-%d-start", rec.CycleID)
		signpostID = stringPtr(startID)
		data := recoveryWindowData(cycle)
		if !cycle.Start.IsZero() {
			if err := w.writeSignpost(hpxSignpostRecord{
				Table:    "signpost",
				Kind:     "recovery_window",
				TS:       formatHPXTimestamp(cycle.Start),
				Edge:     "start",
				Source:   hpxSource,
				OriginID: fmt.Sprintf("whoop-recovery-window-%d-start", rec.CycleID),
				ID:       startID,
				Data:     data,
			}); err != nil {
				return err
			}
		} else {
			signpostID = nil
		}
		if !cycle.End.IsZero() {
			if err := w.writeSignpost(hpxSignpostRecord{
				Table:    "signpost",
				Kind:     "recovery_window",
				TS:       formatHPXTimestamp(cycle.End),
				Edge:     "end",
				Source:   hpxSource,
				OriginID: fmt.Sprintf("whoop-recovery-window-%d-end", rec.CycleID),
				ID:       fmt.Sprintf("recovery-window-%d-end", rec.CycleID),
				Data:     data,
			}); err != nil {
				return err
			}
		}
	}

	date := recoveryMetricDate(rec, cycle)
	if date == "" {
		return nil
	}
	meta := compactMap(map[string]any{
		"cycle_id":         rec.CycleID,
		"sleep_id":         trimToNil(rec.SleepID),
		"score_state":      trimToNil(rec.ScoreState),
		"user_calibrating": rec.Score.UserCalibrating,
		"created_at":       timestampOrNil(rec.CreatedAt),
		"updated_at":       timestampOrNil(rec.UpdatedAt),
	})
	if value := floatPtrValue(rec.Score.RecoveryScore); value != nil {
		if err := w.writeMetric(date, "recovery.score_pct", *value, metricTS, fmt.Sprintf("whoop-recovery-%d-score", rec.CycleID), signpostID, meta); err != nil {
			return err
		}
	}
	if value := floatPtrValue(rec.Score.RestingHeartRate); value != nil {
		if err := w.writeMetric(date, "recovery.resting_hr_bpm", *value, metricTS, fmt.Sprintf("whoop-recovery-%d-resting-hr", rec.CycleID), signpostID, meta); err != nil {
			return err
		}
	}
	if value := floatPtrValue(rec.Score.HRVRMSSDMilli); value != nil {
		if err := w.writeMetric(date, "recovery.hrv_ms", *value, metricTS, fmt.Sprintf("whoop-recovery-%d-hrv", rec.CycleID), signpostID, meta); err != nil {
			return err
		}
	}
	if value := floatPtrValue(rec.Score.Spo2Percentage); value != nil {
		if err := w.writeMetric(date, "recovery.spo2_pct", *value, metricTS, fmt.Sprintf("whoop-recovery-%d-spo2", rec.CycleID), signpostID, meta); err != nil {
			return err
		}
	}
	if value := floatPtrValue(rec.Score.RespiratoryRate); value != nil {
		if err := w.writeMetric(date, "recovery.respiratory_rate_rpm", *value, metricTS, fmt.Sprintf("whoop-recovery-%d-respiratory-rate", rec.CycleID), signpostID, meta); err != nil {
			return err
		}
	}
	if value := floatPtrValue(rec.Score.SkinTempCelsius); value != nil {
		if err := w.writeMetric(date, "recovery.skin_temp_c", *value, metricTS, fmt.Sprintf("whoop-recovery-%d-skin-temp", rec.CycleID), signpostID, meta); err != nil {
			return err
		}
	}
	return nil
}

func (w *hpxWriter) writeCycle(cycle cycles.Cycle) error {
	startID := fmt.Sprintf("recovery-window-%d-start", cycle.ID)
	signpostID := stringPtr(startID)
	metricTS := cycleMetricTimestamp(cycle)
	data := recoveryWindowData(&cycle)
	if !cycle.Start.IsZero() {
		if err := w.writeSignpost(hpxSignpostRecord{
			Table:    "signpost",
			Kind:     "recovery_window",
			TS:       formatHPXTimestamp(cycle.Start),
			Edge:     "start",
			Source:   hpxSource,
			OriginID: fmt.Sprintf("whoop-recovery-window-%d-start", cycle.ID),
			ID:       startID,
			Data:     data,
		}); err != nil {
			return err
		}
	} else {
		signpostID = nil
	}
	if !cycle.End.IsZero() {
		if err := w.writeSignpost(hpxSignpostRecord{
			Table:    "signpost",
			Kind:     "recovery_window",
			TS:       formatHPXTimestamp(cycle.End),
			Edge:     "end",
			Source:   hpxSource,
			OriginID: fmt.Sprintf("whoop-recovery-window-%d-end", cycle.ID),
			ID:       fmt.Sprintf("recovery-window-%d-end", cycle.ID),
			Data:     data,
		}); err != nil {
			return err
		}
	}

	date := metricDateFromTimestamp(cycle.End, cycle.Start)
	if date == "" || cycle.Score.Strain <= 0 {
		return nil
	}
	return w.writeMetric(date, "strain.day_score", cycle.Score.Strain, metricTS, fmt.Sprintf("whoop-cycle-%d-strain", cycle.ID), signpostID, compactMap(map[string]any{
		"cycle_id":           cycle.ID,
		"score_state":        trimToNil(cycle.ScoreState),
		"timezone_offset":    trimToNil(cycle.TimezoneOffset),
		"kilojoule":          floatPtrValue(cycle.Score.Kilojoule),
		"average_heart_rate": intPtrValue(cycle.Score.AverageHeartRate),
		"max_heart_rate":     intPtrValue(cycle.Score.MaxHeartRate),
	}))
}

func (w *hpxWriter) writeProfile(summary *profile.Summary) error {
	if summary == nil || summary.UpdatedAt == nil {
		return nil
	}
	date := summary.UpdatedAt.Format("2006-01-02")
	suffix := summary.UpdatedAt.UTC().Format("20060102T150405Z")
	meta := compactMap(map[string]any{
		"user_id":         trimToNil(summary.UserID),
		"name":            trimToNil(summary.Name),
		"email":           trimToNil(summary.Email),
		"locale":          trimToNil(summary.Locale),
		"timezone":        trimToNil(summary.Timezone),
		"membership_tier": trimToNil(summary.MembershipTier),
		"height_cm":       floatPtrValue(summary.HeightCm),
		"height_in":       floatPtrValue(summary.HeightIn),
		"max_heart_rate":  intPtrValue(summary.MaxHeartRate),
		"updated_at":      summary.UpdatedAt.Format(time.RFC3339),
	})
	if value := floatPtrValue(summary.WeightKg); value != nil {
		if err := w.writeMetric(date, "body.weight_kg", *value, *summary.UpdatedAt, fmt.Sprintf("whoop-profile-%s-weight", suffix), nil, meta); err != nil {
			return err
		}
	}
	if bmi, ok := bmiValue(summary.WeightKg, summary.HeightCm); ok {
		if err := w.writeMetric(date, "body.bmi", bmi, *summary.UpdatedAt, fmt.Sprintf("whoop-profile-%s-bmi", suffix), nil, compactMap(map[string]any{
			"user_id":      trimToNil(summary.UserID),
			"updated_at":   summary.UpdatedAt.Format(time.RFC3339),
			"derived_from": "weight_kg,height_cm",
		})); err != nil {
			return err
		}
	}
	return nil
}

func (w *hpxWriter) writeMetric(date, key string, value float64, ts time.Time, originID string, signpostID *string, meta map[string]any) error {
	return w.write(hpxMetricRecord{
		Table:      "metric",
		Date:       date,
		Key:        key,
		Value:      roundMetricValue(value),
		Source:     hpxSource,
		OriginID:   originID,
		TS:         formatMetricTimestamp(ts),
		SignpostID: signpostID,
		Meta:       meta,
	})
}

func lookupCycle(ctx context.Context, cycleLookup func(context.Context, string) (*cycles.Cycle, error), cycleID int64) *cycles.Cycle {
	if cycleLookup == nil || cycleID == 0 {
		return nil
	}
	cycle, err := cycleLookup(ctx, fmt.Sprintf("%d", cycleID))
	if err != nil {
		return nil
	}
	return cycle
}

func recoveryWindowData(cycle *cycles.Cycle) map[string]any {
	if cycle == nil {
		return nil
	}
	return compactMap(map[string]any{
		"cycle_id":           cycle.ID,
		"timezone_offset":    trimToNil(cycle.TimezoneOffset),
		"score_state":        trimToNil(cycle.ScoreState),
		"kilojoule":          floatPtrValue(cycle.Score.Kilojoule),
		"average_heart_rate": intPtrValue(cycle.Score.AverageHeartRate),
		"max_heart_rate":     intPtrValue(cycle.Score.MaxHeartRate),
	})
}

func sleepMetricDate(sess sleep.Session) string {
	return metricDateFromTimestamp(sleepMetricTimestamp(sess), time.Time{})
}

func recoveryMetricDate(rec recovery.Recovery, cycle *cycles.Cycle) string {
	return metricDateFromTimestamp(recoveryMetricTimestamp(rec, cycle), time.Time{})
}

func metricDateFromTimestamp(primary, fallback time.Time) string {
	if !primary.IsZero() {
		return primary.Format("2006-01-02")
	}
	if !fallback.IsZero() {
		return fallback.Format("2006-01-02")
	}
	return ""
}

func formatHPXTimestamp(ts time.Time) string {
	return ts.Format(time.RFC3339)
}

func formatMetricTimestamp(ts time.Time) string {
	if ts.IsZero() {
		return ""
	}
	return formatHPXTimestamp(ts)
}

func sleepMetricTimestamp(sess sleep.Session) time.Time {
	return firstTimestamp(sess.End, sess.Start, sess.CreatedAt, sess.UpdatedAt)
}

func workoutMetricTimestamp(workout workouts.Workout) time.Time {
	return firstTimestamp(workout.Start, workout.End, workout.CreatedAt, workout.UpdatedAt)
}

func recoveryMetricTimestamp(rec recovery.Recovery, cycle *cycles.Cycle) time.Time {
	if ts := firstTimestamp(rec.CreatedAt, rec.UpdatedAt); !ts.IsZero() {
		return ts
	}
	if cycle == nil {
		return time.Time{}
	}
	return cycleMetricTimestamp(*cycle)
}

func cycleMetricTimestamp(cycle cycles.Cycle) time.Time {
	return firstTimestamp(cycle.End, cycle.Start, cycle.UpdatedAt, cycle.CreatedAt)
}

func firstTimestamp(values ...time.Time) time.Time {
	for _, value := range values {
		if !value.IsZero() {
			return value
		}
	}
	return time.Time{}
}

func durationMillis(start, end time.Time) (float64, bool) {
	if start.IsZero() || end.IsZero() || !end.After(start) {
		return 0, false
	}
	return float64(end.Sub(start).Milliseconds()), true
}

func bmiValue(weightKg, heightCm *float64) (float64, bool) {
	if weightKg == nil || heightCm == nil || *heightCm <= 0 {
		return 0, false
	}
	heightMeters := *heightCm / 100
	if heightMeters <= 0 {
		return 0, false
	}
	return *weightKg / (heightMeters * heightMeters), true
}

func floatPtrValue(value *float64) *float64 {
	if value == nil {
		return nil
	}
	v := *value
	return &v
}

func intPtrValue(value *int) *float64 {
	if value == nil {
		return nil
	}
	v := float64(*value)
	return &v
}

func int64PtrValue(value *int64) *float64 {
	if value == nil {
		return nil
	}
	v := float64(*value)
	return &v
}

func nonZeroInt64(value int64) *float64 {
	if value == 0 {
		return nil
	}
	v := float64(value)
	return &v
}

func timestampOrNil(ts time.Time) *string {
	if ts.IsZero() {
		return nil
	}
	formatted := ts.Format(time.RFC3339)
	return &formatted
}

func trimToNil(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func stringPtr(value string) *string {
	return &value
}

func compactMap(values map[string]any) map[string]any {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]any, len(values))
	for key, value := range values {
		switch v := value.(type) {
		case nil:
			continue
		case *string:
			if v == nil || strings.TrimSpace(*v) == "" {
				continue
			}
			out[key] = *v
		case string:
			if strings.TrimSpace(v) == "" {
				continue
			}
			out[key] = v
		case *float64:
			if v == nil {
				continue
			}
			out[key] = roundMetricValue(*v)
		case float64:
			out[key] = roundMetricValue(v)
		case bool:
			out[key] = v
		default:
			out[key] = value
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func roundMetricValue(value float64) float64 {
	return math.Round(value*1000) / 1000
}
