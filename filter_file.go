package loges

import (
	"encoding/json"

	u "github.com/araddon/gou"
)

var expectedLevels = map[string]bool{
	"DEBU":   true,
	"DEBG":   true,
	"DEBUG":  true,
	"INFO":   true,
	"ERROR":  true,
	"ERRO":   true,
	"WARN":   true,
	"FATAL":  true,
	"FATA":   true,
	"METRIC": true,
	"METR":   true,
}

// file format [date source jsonmessage] parser
func FileFormatter(logstashType string, tags []string) LineTransform {
	return func(d *LineEvent) *Event {

		//u.Infof("%v line event: %v  Metric?%v  json?%v", d.Ts, d.LogLevel, d.IsMetric(), d.IsJson())

		// Don't log out metrics
		if d.IsMetric() {
			return nil
		}
		if len(d.Data) < 10 {
			u.Warn("Invalid line?", string(d.Data))
			return nil
		} else if !d.Ts.IsZero() {
			if d.IsJson() {
				evt := NewTsEvent(logstashType, d.Source, string(d.Data), d.Ts)
				evt.Fields = make(map[string]interface{})
				evt.Fields["codefile"] = d.Prefix
				evt.Fields["host"] = hostName
				evt.Fields["level"] = d.LogLevel
				evt.Fields["WriteErrs"] = d.WriteErrs
				jd := json.RawMessage(d.Data)
				m := make(map[string]interface{})
				if err := json.Unmarshal(d.Data, &m); err == nil {
					evt.Raw = &jd
				}
				return evt
			}
			evt := NewTsEvent(logstashType, d.Source, d.Prefix+" "+string(d.Data), d.Ts)
			evt.Fields = make(map[string]interface{})
			evt.Fields["host"] = hostName
			evt.Fields["codefile"] = d.Prefix
			evt.Fields["level"] = d.LogLevel
			evt.Fields["WriteErrs"] = d.WriteErrs
			return evt

		}
		return nil
	}
}
