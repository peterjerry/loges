package loges

import (
	"bytes"
	"io/ioutil"
	"strings"
	"time"

	"github.com/araddon/dateparse"
	u "github.com/araddon/gou"
)

// This formatter reads go files and performs:
//  1.  Squashes multiple lines into one (as needed), Tries to squash panics(go) into one line
//  2.  Reads out the LineType/Level [DEBUG,INFO,METRIC] into a field
//
// This expects log files in this format
//   2013-05-25 13:25:32.475 authctx.go:169: [DEBUG] sink       Building sink for kafka from factory method
func MakeFileFlattener(filename string, msgChan chan *LineEvent) func(string) {
	// Builder used to build the colored string.
	buf := new(bytes.Buffer)

	startsDate := false
	prevWasDate := false
	pos := 0
	posEnd := 0
	var dataType []byte
	var loglevel string
	var dateStr, prevDateStr string
	var prevLogTs time.Time
	lineCt := 0

	return func(line string) {
		lineCt++
		if len(line) < 8 {
			buf.WriteString(line)
			return
		}

		startsDate = false
		spaceCt := 0

		//    [DATE]                  [SOURCE]              [LEVEL] [MESSAGE]
		// 2014/07/10 11:04:20.653185 filter_fluentd.go:16: [DEBUG] %s
		for i := 0; i < len(line); i++ {
			r := line[i]
			if r == ' ' {
				if spaceCt == 1 {
					dateStr = string(line[:i])
					if dts, err := dateparse.ParseAny(dateStr); err == nil {
						startsDate = true
						defer func() {
							// defer will run after prevDateStr already used to send message
							prevLogTs = dts
							prevDateStr = dateStr
						}()
					}
					break
				}
				spaceCt++
			}
		}

		// Find first square bracket wrapper:   [WARN]
		// 2014/07/10 11:04:20.653185 filter_fluentd.go:16: [DEBUG] %s
		// datestr                                         pos, posEnd
		pos = strings.IndexRune(line, '[')
		posEnd = strings.IndexRune(line, ']')
		if pos > 0 && posEnd > 0 && pos < posEnd && len(line) > pos && len(line) > posEnd {
			loglevel = line[pos+1 : posEnd]
			// If we don't find, it probably wasn't one of [INFO],[WARN] etc so accumulate
			if _, ok := expectedLevels[loglevel]; !ok {
				buf.WriteString(line)
				return
			}
		}

		//u.Debugf("pos=%d datatype=%s num?=%v", pos, dataType, startsDate)
		//u.Infof("starts with date?=%v prev?%v pos=%d lvl=%s short[]%v len=%d buf.len=%d", startsDate, prevWasDate, pos, loglevel, (posEnd-pos) < 8, len(line), buf.Len())
		if pos == -1 && !prevWasDate {
			// accumulate in buffer, probably/possibly a panic?
			buf.WriteString(line)
			buf.WriteString(" \n")
		} else if !startsDate {
			// accumulate in buffer
			buf.WriteString(line)
			buf.WriteString(" \n")
		} else if posEnd-8 > pos {
			// position of [block]  too long, so ignore
			buf.WriteString(line)
			buf.WriteString(" \n")
		} else if pos > 80 {
			// [WARN] should be at beginning of line
			buf.WriteString(line)
			buf.WriteString(" \n")
		} else {

			// Line had [LEVEL] AND startsDate at start so go ahead and log it

			if buf.Len() == 0 {
				// lets buffer it, ensuring we have the completion of this line
				buf.WriteString(line)
				return
			}

			// we already have previous line in buffer
			data, err := ioutil.ReadAll(buf)
			if err == nil {
				pos = bytes.IndexRune(data, '[')
				posEnd = bytes.IndexRune(data, ']')
				preFix := ""
				if posEnd-8 > pos {
					//u.Warnf("level:%s  \n\nline=%s", string(data[pos+1:posEnd]), string(data))
					//buf.WriteString(line)
					return
				} else if pos > 0 && posEnd > 0 && pos < posEnd && len(data) > pos && len(data) > posEnd {
					dataType = data[pos+1 : posEnd]
					if len(data) > len(prevDateStr) {
						preFix = string(data[len(prevDateStr)+1 : posEnd])
						//                            [prefix             |- posEnd
						// 2016/09/14 02:33:01.465711 entity.go:179: [ERROR]
						preFixParts := strings.Split(preFix, ": ")
						if len(preFixParts) > 1 {
							preFix = preFixParts[0]
						}
						data = data[posEnd+1:]

					}

				} else {
					dataType = []byte("NA")
					//u.Warnf("level:%s  \n\nline=%s", string(data[pos+1:posEnd]), string(data))
				}
				// if !bytes.HasPrefix(data, datePrefix) {
				// 	u.Warnf("ct=%d level:%s  \n\nline=%s", lineCt, string(data[pos+1:posEnd]), string(data))
				// }
				le := LineEvent{Data: data, Prefix: preFix, Ts: prevLogTs, LogLevel: string(dataType), Source: filename, WriteErrs: 0}
				//u.Debugf("lineevent: %+v", le)
				msgChan <- &le

			} else {
				u.Error(err)
			}
			// now write this line for next analysis
			buf.WriteString(line)
		}
		prevWasDate = startsDate
	}
}
