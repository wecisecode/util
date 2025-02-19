// Package dateparse parses date-strings without knowing the format
// in advance, using a fast lex based approach to eliminate shotgun
// attempts.  It leans towards US style dates when there is a conflict.
package dateparse

import (
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// func init() {
// 	gou.SetupLogging("debug")
// 	gou.SetColorOutput()
// }

var months = []string{
	"january",
	"february",
	"march",
	"april",
	"may",
	"june",
	"july",
	"august",
	"september",
	"october",
	"november",
	"december",
}

type dateState uint8
type timeState uint8

const (
	dateStart dateState = iota // 0
	dateDigit
	dateYearDash
	dateYearDashAlphaDash
	dateYearDashDash
	dateYearDashDashWs // 5
	dateYearDashDashT
	dateDigitDash
	dateDigitDashAlpha
	dateDigitDashAlphaDash
	dateDigitDot // 10
	dateDigitDotDot
	dateDigitSlash
	dateDigitChineseYear
	dateDigitChineseYearWs
	dateDigitWs // 15
	dateDigitWsMoYear
	dateDigitWsMolong
	dateAlpha
	dateAlphaWs
	dateAlphaWsDigit // 20
	dateAlphaWsDigitMore
	dateAlphaWsDigitMoreWs
	dateAlphaWsDigitMoreWsYear
	dateAlphaWsMonth
	dateAlphaWsMonthMore
	dateAlphaWsMonthSuffix
	dateAlphaWsMore
	dateAlphaWsAtTime
	dateAlphaWsAlpha
	dateAlphaWsAlphaYearmaybe
	dateAlphaPeriodWsDigit
	dateWeekdayComma
	dateWeekdayAbbrevComma
)
const (
	// Time state
	timeIgnore timeState = iota // 0
	timeStart
	timeWs
	timeWsAlpha
	timeWsAlphaWs
	timeWsAlphaZoneOffset // 5
	timeWsAlphaZoneOffsetWs
	timeWsAlphaZoneOffsetWsYear
	timeWsAlphaZoneOffsetWsExtra
	timeWsAMPMMaybe
	timeWsAMPM // 10
	timeWsOffset
	timeWsOffsetWs // 12
	timeWsOffsetColonAlpha
	timeWsOffsetColon
	timeWsYear // 15
	timeOffset
	timeOffsetColon
	timeAlpha
	timePeriod
	timePeriodOffset // 20
	timePeriodOffsetColon
	timePeriodOffsetColonWs
	timePeriodWs
	timePeriodWsAlpha
	timePeriodWsOffset // 25
	timePeriodWsOffsetWs
	timePeriodWsOffsetWsAlpha
	timePeriodWsOffsetColon
	timePeriodWsOffsetColonAlpha
	timeZ
	timeZDigit
)

var (
	// ErrAmbiguousMMDD for date formats such as 04/02/2014 the mm/dd vs dd/mm are
	// ambiguous, so it is an error for strict parse rules.
	ErrAmbiguousMMDD = fmt.Errorf("This date has ambiguous mm/dd vs dd/mm type format")
)

func unknownErr(datestr string) error {
	return fmt.Errorf("Could not find format for %q", datestr)
}

// ParseAny parse an unknown date format, detect the layout.
// Normal parse.  Equivalent Timezone rules as time.Parse().
// NOTE:  please see readme on mmdd vs ddmm ambiguous dates.
func ParseAny(datestr string) (time.Time, error) {
	p, err := ParseTime(datestr, nil)
	if err != nil {
		return time.Time{}, err
	}
	return p.Parse()
}

// ParseIn with Location, equivalent to time.ParseInLocation() timezone/offset
// rules.  Using location arg, if timezone/offset info exists in the
// datestring, it uses the given location rules for any zone interpretation.
// That is, MST means one thing when using America/Denver and something else
// in other locations.
func ParseIn(datestr string, loc *time.Location) (time.Time, error) {
	p, err := ParseTime(datestr, loc)
	if err != nil {
		return time.Time{}, err
	}
	return p.Parse()
}

// ParseLocal Given an unknown date format, detect the layout,
// using time.Local, parse.
//
// Set Location to time.Local.  Same as ParseIn Location but lazily uses
// the global time.Local variable for Location argument.
//
//     denverLoc, _ := time.LoadLocation("America/Denver")
//     time.Local = denverLoc
//
//     t, err := dateparse.ParseLocal("3/1/2014")
//
// Equivalent to:
//
//     t, err := dateparse.ParseIn("3/1/2014", denverLoc)
//
func ParseLocal(datestr string) (time.Time, error) {
	p, err := ParseTime(datestr, time.Local)
	if err != nil {
		return time.Time{}, err
	}
	return p.Parse()
}

// MustParse  parse a date, and panic if it can't be parsed.  Used for testing.
// Not recommended for most use-cases.
func MustParse(datestr string) time.Time {
	p, err := ParseTime(datestr, nil)
	if err != nil {
		panic(err.Error())
	}
	t, err := p.Parse()
	if err != nil {
		panic(err.Error())
	}
	return t
}

// ParseFormat parse's an unknown date-time string and returns a layout
// string that can parse this (and exact same format) other date-time strings.
//
//     layout, err := dateparse.ParseFormat("2013-02-01 00:00:00")
//     // layout = "2006-01-02 15:04:05"
//
func ParseFormat(datestr string) (string, error) {
	p, err := ParseTime(datestr, nil)
	if err != nil {
		return "", err
	}
	_, err = p.Parse()
	if err != nil {
		return "", err
	}
	return string(p.Format), nil
}

// ParseStrict parse an unknown date format.  IF the date is ambigous
// mm/dd vs dd/mm then return an error. These return errors:   3.3.2014 , 8/8/71 etc
func ParseStrict(datestr string) (time.Time, error) {
	p, err := ParseTime(datestr, nil)
	if err != nil {
		return time.Time{}, err
	}
	if p.ambiguousMD {
		return time.Time{}, ErrAmbiguousMMDD
	}
	return p.Parse()
}

func ParseTime(datestr string, loc *time.Location) (*parser, error) {

	p := newParser(datestr, loc)
	i := 0

	// General strategy is to read rune by rune through the date looking for
	// certain hints of what type of date we are dealing with.
	// Hopefully we only need to read about 5 or 6 bytes before
	// we figure it out and then attempt a parse
iterRunes:
	for ; i < len(datestr); i++ {
		//r := rune(datestr[i])
		r, bytesConsumed := utf8.DecodeRuneInString(datestr[i:])
		if bytesConsumed > 1 {
			i += (bytesConsumed - 1)
		}

		//gou.Debugf("i=%d r=%s state=%d   %s", i, string(r), p.stateDate, datestr)
		switch p.stateDate {
		case dateStart:
			if unicode.IsDigit(r) {
				p.stateDate = dateDigit
			} else if unicode.IsLetter(r) {
				p.stateDate = dateAlpha
			} else {
				return nil, unknownErr(datestr)
			}
		case dateDigit:

			switch r {
			case '-', '\u2212':
				// 2006-01-02
				// 2013-Feb-03
				// 13-Feb-03
				// 29-Jun-2016
				if i == 4 {
					p.stateDate = dateYearDash
					p.yeari = 0
					p.yearlen = i
					p.moi = i + 1
					p.set(0, "2006")
				} else {
					p.stateDate = dateDigitDash
				}
			case '/':
				// 03/31/2005
				// 2014/02/24
				p.stateDate = dateDigitSlash
				if i == 4 {
					p.yearlen = i
					p.moi = i + 1
					p.setYear()
				} else {
					p.ambiguousMD = true
					if p.preferMonthFirst {
						if p.molen == 0 {
							p.molen = i
							p.setMonth()
							p.dayi = i + 1
						}
					}
				}

			case '.':
				// 3.31.2014
				// 08.21.71
				// 2014.05
				p.stateDate = dateDigitDot
				if i == 4 {
					p.yearlen = i
					p.moi = i + 1
					p.setYear()
				} else {
					p.ambiguousMD = true
					p.moi = 0
					p.molen = i
					p.setMonth()
					p.dayi = i + 1
				}

			case ' ':
				// 18 January 2018
				// 8 January 2018
				// 8 jan 2018
				// 02 Jan 2018 23:59
				// 02 Jan 2018 23:59:34
				// 12 Feb 2006, 19:17
				// 12 Feb 2006, 19:17:22
				p.stateDate = dateDigitWs
				p.dayi = 0
				p.daylen = i
			case '年':
				// Chinese Year
				p.stateDate = dateDigitChineseYear
			case ',':
				return nil, unknownErr(datestr)
			default:
				continue
			}
			p.part1Len = i

		case dateYearDash:
			// dateYearDashDashT
			//  2006-01-02T15:04:05Z07:00
			// dateYearDashDashWs
			//  2013-04-01 22:43:22
			// dateYearDashAlphaDash
			//   2013-Feb-03
			switch r {
			case '-':
				p.molen = i - p.moi
				p.dayi = i + 1
				p.stateDate = dateYearDashDash
				p.setMonth()
			default:
				if unicode.IsLetter(r) {
					p.stateDate = dateYearDashAlphaDash
				}
			}

		case dateYearDashDash:
			// dateYearDashDashT
			//  2006-01-02T15:04:05Z07:00
			// dateYearDashDashWs
			//  2013-04-01 22:43:22
			switch r {
			case ' ':
				p.daylen = i - p.dayi
				p.stateDate = dateYearDashDashWs
				p.stateTime = timeStart
				p.setDay()
				break iterRunes
			case 'T':
				p.daylen = i - p.dayi
				p.stateDate = dateYearDashDashT
				p.stateTime = timeStart
				p.setDay()
				break iterRunes
			}
		case dateYearDashAlphaDash:
			// 2013-Feb-03
			switch r {
			case '-':
				p.molen = i - p.moi
				p.set(p.moi, "Jan")
				p.dayi = i + 1
			}
		case dateDigitDash:
			// 13-Feb-03
			// 29-Jun-2016
			if unicode.IsLetter(r) {
				p.stateDate = dateDigitDashAlpha
				p.moi = i
			} else {
				return nil, unknownErr(datestr)
			}
		case dateDigitDashAlpha:
			// 13-Feb-03
			// 28-Feb-03
			// 29-Jun-2016
			switch r {
			case '-':
				p.molen = i - p.moi
				p.set(p.moi, "Jan")
				p.yeari = i + 1
				p.stateDate = dateDigitDashAlphaDash
			}

		case dateDigitDashAlphaDash:
			// 13-Feb-03   ambiguous
			// 28-Feb-03   ambiguous
			// 29-Jun-2016
			switch r {
			case ' ':
				// we need to find if this was 4 digits, aka year
				// or 2 digits which makes it ambiguous year/day
				length := i - (p.moi + p.molen + 1)
				if length == 4 {
					p.yearlen = 4
					p.set(p.yeari, "2006")
					// We now also know that part1 was the day
					p.dayi = 0
					p.daylen = p.part1Len
					p.setDay()
				} else if length == 2 {
					// We have no idea if this is
					// yy-mon-dd   OR  dd-mon-yy
					//
					// We are going to ASSUME (bad, bad) that it is dd-mon-yy  which is a horible assumption
					p.ambiguousMD = true
					p.yearlen = 2
					p.set(p.yeari, "06")
					// We now also know that part1 was the day
					p.dayi = 0
					p.daylen = p.part1Len
					p.setDay()
				}
				p.stateTime = timeStart
				break iterRunes
			}

		case dateDigitSlash:
			// 2014/07/10 06:55:38.156283
			// 03/19/2012 10:11:59
			// 04/2/2014 03:00:37
			// 3/1/2012 10:11:59
			// 4/8/2014 22:05
			// 3/1/2014
			// 10/13/2014
			// 01/02/2006
			// 1/2/06

			switch r {
			case ' ':
				p.stateTime = timeStart
				if p.yearlen == 0 {
					p.yearlen = i - p.yeari
					p.setYear()
				} else if p.daylen == 0 {
					p.daylen = i - p.dayi
					p.setDay()
				}
				break iterRunes
			case '/':
				if p.yearlen > 0 {
					// 2014/07/10 06:55:38.156283
					if p.molen == 0 {
						p.molen = i - p.moi
						p.setMonth()
						p.dayi = i + 1
					}
				} else if p.preferMonthFirst {
					if p.daylen == 0 {
						p.daylen = i - p.dayi
						p.setDay()
						p.yeari = i + 1
					}
				}
			}

		case dateDigitWs:
			// 18 January 2018
			// 8 January 2018
			// 8 jan 2018
			// 1 jan 18
			// 02 Jan 2018 23:59
			// 02 Jan 2018 23:59:34
			// 12 Feb 2006, 19:17
			// 12 Feb 2006, 19:17:22
			switch r {
			case ' ':
				p.yeari = i + 1
				//p.yearlen = 4
				p.dayi = 0
				p.daylen = p.part1Len
				p.setDay()
				p.stateTime = timeStart
				if i > p.daylen+len(" Sep") { //  November etc
					// If len greather than space + 3 it must be full month
					p.stateDate = dateDigitWsMolong
				} else {
					// If len=3, the might be Feb or May?  Ie ambigous abbreviated but
					// we can parse may with either.  BUT, that means the
					// format may not be correct?
					// mo := strings.ToLower(datestr[p.daylen+1 : i])
					p.moi = p.daylen + 1
					p.molen = i - p.moi
					p.set(p.moi, "Jan")
					p.stateDate = dateDigitWsMoYear
				}
			}

		case dateDigitWsMoYear:
			// 8 jan 2018
			// 02 Jan 2018 23:59
			// 02 Jan 2018 23:59:34
			// 12 Feb 2006, 19:17
			// 12 Feb 2006, 19:17:22
			switch r {
			case ',':
				p.yearlen = i - p.yeari
				p.setYear()
				i++
				break iterRunes
			case ' ':
				p.yearlen = i - p.yeari
				p.setYear()
				break iterRunes
			}
		case dateDigitWsMolong:
			// 18 January 2018
			// 8 January 2018

		case dateDigitChineseYear:
			// dateDigitChineseYear
			//   2014年04月08日
			//               weekday  %Y年%m月%e日 %A %I:%M %p
			// 2013年07月18日 星期四 10:27 上午
			if r == ' ' {
				p.stateDate = dateDigitChineseYearWs
				break
			}
		case dateDigitDot:
			// This is the 2nd period
			// 3.31.2014
			// 08.21.71
			// 2014.05
			// 2018.09.30
			if r == '.' {
				if p.moi == 0 {
					// 3.31.2014
					p.daylen = i - p.dayi
					p.yeari = i + 1
					p.setDay()
					p.stateDate = dateDigitDotDot
				} else {
					// 2018.09.30
					//p.molen = 2
					p.molen = i - p.moi
					p.dayi = i + 1
					p.setMonth()
					p.stateDate = dateDigitDotDot
				}
			}
		case dateDigitDotDot:
			// iterate all the way through
		case dateAlpha:
			// dateAlphaWS
			//  Mon Jan _2 15:04:05 2006
			//  Mon Jan _2 15:04:05 MST 2006
			//  Mon Jan 02 15:04:05 -0700 2006
			//  Mon Aug 10 15:44:11 UTC+0100 2015
			//  Fri Jul 03 2015 18:04:07 GMT+0100 (GMT Daylight Time)
			//  dateAlphaWSDigit
			//    May 8, 2009 5:57:51 PM
			//    oct 1, 1970
			//  dateAlphaWsMonth
			//    April 8, 2009
			//  dateAlphaWsMore
			//    dateAlphaWsAtTime
			//      January 02, 2006 at 3:04pm MST-07
			//
			//  dateAlphaPeriodWsDigit
			//    oct. 1, 1970
			// dateWeekdayComma
			//   Monday, 02 Jan 2006 15:04:05 MST
			//   Monday, 02-Jan-06 15:04:05 MST
			//   Monday, 02 Jan 2006 15:04:05 -0700
			//   Monday, 02 Jan 2006 15:04:05 +0100
			// dateWeekdayAbbrevComma
			//   Mon, 02 Jan 2006 15:04:05 MST
			//   Mon, 02 Jan 2006 15:04:05 -0700
			//   Thu, 13 Jul 2017 08:58:40 +0100
			//   Tue, 11 Jul 2017 16:28:13 +0200 (CEST)
			//   Mon, 02-Jan-06 15:04:05 MST
			switch {
			case r == ' ':
				//      X
				// April 8, 2009
				if i > 3 {
					// Check to see if the alpha is name of month?  or Day?
					month := strings.ToLower(datestr[0:i])
					if isMonthFull(month) {
						p.fullMonth = month
						// len(" 31, 2018")   = 9
						if len(datestr[i:]) < 10 {
							// April 8, 2009
							p.stateDate = dateAlphaWsMonth
						} else {
							p.stateDate = dateAlphaWsMore
						}
						p.dayi = i + 1
						break
					}

				} else {
					// This is possibly ambiguous?  May will parse as either though.
					// So, it could return in-correct format.
					// May 05, 2005, 05:05:05
					// May 05 2005, 05:05:05
					// Jul 05, 2005, 05:05:05
					p.stateDate = dateAlphaWs
				}

			case r == ',':
				// Mon, 02 Jan 2006
				// p.moi = 0
				// p.molen = i
				if i == 3 {
					p.stateDate = dateWeekdayAbbrevComma
					p.set(0, "Mon")
				} else {
					p.stateDate = dateWeekdayComma
					p.skip = i + 2
					i++
					// TODO:  lets just make this "skip" as we don't need
					// the mon, monday, they are all superfelous and not needed
					// just lay down the skip, no need to fill and then skip
				}
			case r == '.':
				// sept. 28, 2017
				// jan. 28, 2017
				p.stateDate = dateAlphaPeriodWsDigit
				if i == 3 {
					p.molen = i
					p.set(0, "Jan")
				} else if i == 4 {
					// gross
					datestr = datestr[0:i-1] + datestr[i:]
					return ParseTime(datestr, loc)
				} else {
					return nil, unknownErr(datestr)
				}
			}

		case dateAlphaWs:
			// dateAlphaWsAlpha
			//   Mon Jan _2 15:04:05 2006
			//   Mon Jan _2 15:04:05 MST 2006
			//   Mon Jan 02 15:04:05 -0700 2006
			//   Fri Jul 03 2015 18:04:07 GMT+0100 (GMT Daylight Time)
			//   Mon Aug 10 15:44:11 UTC+0100 2015
			//  dateAlphaWsDigit
			//    May 8, 2009 5:57:51 PM
			//    May 8 2009 5:57:51 PM
			//    oct 1, 1970
			//    oct 7, '70
			switch {
			case unicode.IsLetter(r):
				p.set(0, "Mon")
				p.stateDate = dateAlphaWsAlpha
				p.set(i, "Jan")
			case unicode.IsDigit(r):
				p.set(0, "Jan")
				p.stateDate = dateAlphaWsDigit
				p.dayi = i
			}

		case dateAlphaWsDigit:
			// May 8, 2009 5:57:51 PM
			// May 8 2009 5:57:51 PM
			// oct 1, 1970
			// oct 7, '70
			// oct. 7, 1970
			if r == ',' {
				p.daylen = i - p.dayi
				p.setDay()
				p.stateDate = dateAlphaWsDigitMore
			} else if r == ' ' {
				p.daylen = i - p.dayi
				p.setDay()
				p.yeari = i + 1
				p.stateDate = dateAlphaWsDigitMoreWs
			} else if unicode.IsLetter(r) {
				p.stateDate = dateAlphaWsMonthSuffix
				i--
			}
		case dateAlphaWsDigitMore:
			//       x
			// May 8, 2009 5:57:51 PM
			// May 05, 2005, 05:05:05
			// May 05 2005, 05:05:05
			// oct 1, 1970
			// oct 7, '70
			if r == ' ' {
				p.yeari = i + 1
				p.stateDate = dateAlphaWsDigitMoreWs
			}
		case dateAlphaWsDigitMoreWs:
			//            x
			// May 8, 2009 5:57:51 PM
			// May 05, 2005, 05:05:05
			// oct 1, 1970
			// oct 7, '70
			switch r {
			case '\'':
				p.yeari = i + 1
			case ' ', ',':
				//            x
				// May 8, 2009 5:57:51 PM
				//            x
				// May 8, 2009, 5:57:51 PM
				p.stateDate = dateAlphaWsDigitMoreWsYear
				p.yearlen = i - p.yeari
				p.setYear()
				p.stateTime = timeStart
				break iterRunes
			}

		case dateAlphaWsAlpha:
			// Mon Jan _2 15:04:05 2006
			// Mon Jan 02 15:04:05 -0700 2006
			// Mon Jan _2 15:04:05 MST 2006
			// Mon Aug 10 15:44:11 UTC+0100 2015
			// Fri Jul 03 2015 18:04:07 GMT+0100 (GMT Daylight Time)
			if r == ' ' {
				if p.dayi > 0 {
					p.daylen = i - p.dayi
					p.setDay()
					p.yeari = i + 1
					p.stateDate = dateAlphaWsAlphaYearmaybe
					p.stateTime = timeStart
				}
			} else if unicode.IsDigit(r) {
				if p.dayi == 0 {
					p.dayi = i
				}
			}

		case dateAlphaWsAlphaYearmaybe:
			//            x
			// Mon Jan _2 15:04:05 2006
			// Fri Jul 03 2015 18:04:07 GMT+0100 (GMT Daylight Time)
			if r == ':' {
				i = i - 3
				p.stateDate = dateAlphaWsAlpha
				p.yeari = 0
				break iterRunes
			} else if r == ' ' {
				// must be year format, not 15:04
				p.yearlen = i - p.yeari
				p.setYear()
				break iterRunes
			}

		case dateAlphaWsMonth:
			// April 8, 2009
			// April 8 2009
			switch r {
			case ' ', ',':
				//       x
				// June 8, 2009
				//       x
				// June 8 2009
				if p.daylen == 0 {
					p.daylen = i - p.dayi
					p.setDay()
				}
			case 's', 'S', 'r', 'R', 't', 'T', 'n', 'N':
				// st, rd, nd, st
				i--
				p.stateDate = dateAlphaWsMonthSuffix
			default:
				if p.daylen > 0 && p.yeari == 0 {
					p.yeari = i
				}
			}
		case dateAlphaWsMonthMore:
			//                  X
			// January 02, 2006, 15:04:05
			// January 02 2006, 15:04:05
			// January 02, 2006 15:04:05
			// January 02 2006 15:04:05
			switch r {
			case ',':
				p.yearlen = i - p.yeari
				p.setYear()
				p.stateTime = timeStart
				i++
				break iterRunes
			case ' ':
				p.yearlen = i - p.yeari
				p.setYear()
				p.stateTime = timeStart
				break iterRunes
			}
		case dateAlphaWsMonthSuffix:
			//        x
			// April 8th, 2009
			// April 8th 2009
			switch r {
			case 't', 'T':
				if p.nextIs(i, 'h') || p.nextIs(i, 'H') {
					if len(datestr) > i+2 {
						return ParseTime(fmt.Sprintf("%s%s", p.datestr[0:i], p.datestr[i+2:]), loc)
					}
				}
			case 'n', 'N':
				if p.nextIs(i, 'd') || p.nextIs(i, 'D') {
					if len(datestr) > i+2 {
						return ParseTime(fmt.Sprintf("%s%s", p.datestr[0:i], p.datestr[i+2:]), loc)
					}
				}
			case 's', 'S':
				if p.nextIs(i, 't') || p.nextIs(i, 'T') {
					if len(datestr) > i+2 {
						return ParseTime(fmt.Sprintf("%s%s", p.datestr[0:i], p.datestr[i+2:]), loc)
					}
				}
			case 'r', 'R':
				if p.nextIs(i, 'd') || p.nextIs(i, 'D') {
					if len(datestr) > i+2 {
						return ParseTime(fmt.Sprintf("%s%s", p.datestr[0:i], p.datestr[i+2:]), loc)
					}
				}
			}
		case dateAlphaWsMore:
			// January 02, 2006, 15:04:05
			// January 02 2006, 15:04:05
			// January 2nd, 2006, 15:04:05
			// January 2nd 2006, 15:04:05
			// September 17, 2012 at 5:00pm UTC-05
			switch {
			case r == ',':
				//           x
				// January 02, 2006, 15:04:05
				if p.nextIs(i, ' ') {
					p.daylen = i - p.dayi
					p.setDay()
					p.yeari = i + 2
					p.stateDate = dateAlphaWsMonthMore
					i++
				}

			case r == ' ':
				//           x
				// January 02 2006, 15:04:05
				p.daylen = i - p.dayi
				p.setDay()
				p.yeari = i + 1
				p.stateDate = dateAlphaWsMonthMore
			case unicode.IsDigit(r):
				//         XX
				// January 02, 2006, 15:04:05
				continue
			case unicode.IsLetter(r):
				//          X
				// January 2nd, 2006, 15:04:05
				p.daylen = i - p.dayi
				p.setDay()
				p.stateDate = dateAlphaWsMonthSuffix
				i--
			}

		case dateAlphaPeriodWsDigit:
			//    oct. 7, '70
			switch {
			case r == ' ':
				// continue
			case unicode.IsDigit(r):
				p.stateDate = dateAlphaWsDigit
				p.dayi = i
			default:
				return p, unknownErr(datestr)
			}
		case dateWeekdayComma:
			// Monday, 02 Jan 2006 15:04:05 MST
			// Monday, 02 Jan 2006 15:04:05 -0700
			// Monday, 02 Jan 2006 15:04:05 +0100
			// Monday, 02-Jan-06 15:04:05 MST
			if p.dayi == 0 {
				p.dayi = i
			}
			switch r {
			case ' ', '-':
				if p.moi == 0 {
					p.moi = i + 1
					p.daylen = i - p.dayi
					p.setDay()
				} else if p.yeari == 0 {
					p.yeari = i + 1
					p.molen = i - p.moi
					p.set(p.moi, "Jan")
				} else {
					p.stateTime = timeStart
					break iterRunes
				}
			}
		case dateWeekdayAbbrevComma:
			// Mon, 02 Jan 2006 15:04:05 MST
			// Mon, 02 Jan 2006 15:04:05 -0700
			// Thu, 13 Jul 2017 08:58:40 +0100
			// Thu, 4 Jan 2018 17:53:36 +0000
			// Tue, 11 Jul 2017 16:28:13 +0200 (CEST)
			// Mon, 02-Jan-06 15:04:05 MST
			switch r {
			case ' ', '-':
				if p.dayi == 0 {
					p.dayi = i + 1
				} else if p.moi == 0 {
					p.daylen = i - p.dayi
					p.setDay()
					p.moi = i + 1
				} else if p.yeari == 0 {
					p.molen = i - p.moi
					p.set(p.moi, "Jan")
					p.yeari = i + 1
				} else {
					p.yearlen = i - p.yeari
					p.setYear()
					p.stateTime = timeStart
					break iterRunes
				}
			}

		default:
			break iterRunes
		}
	}
	p.coalesceDate(i)
	if p.stateTime == timeStart {
		// increment first one, since the i++ occurs at end of loop
		if i < len(p.datestr) {
			i++
		}
		// ensure we skip any whitespace prefix
		for ; i < len(datestr); i++ {
			r := rune(datestr[i])
			if r != ' ' {
				break
			}
		}

	iterTimeRunes:
		for ; i < len(datestr); i++ {
			r := rune(datestr[i])

			//gou.Debugf("%d %s %d iterTimeRunes  %s %s", i, string(r), p.stateTime, p.ds(), p.ts())

			switch p.stateTime {
			case timeStart:
				// 22:43:22
				// 22:43
				// timeComma
				//   08:20:13,787
				// timeWs
				//   05:24:37 PM
				//   06:20:00 UTC
				//   06:20:00 UTC-05
				//   00:12:00 +0000 UTC
				//   22:18:00 +0000 UTC m=+0.000000001
				//   15:04:05 -0700
				//   15:04:05 -07:00
				//   15:04:05 2008
				// timeOffset
				//   03:21:51+00:00
				//   19:55:00+0100
				// timePeriod
				//   17:24:37.3186369
				//   00:07:31.945167
				//   18:31:59.257000000
				//   00:00:00.000
				//   timePeriodOffset
				//     19:55:00.799+0100
				//     timePeriodOffsetColon
				//       15:04:05.999-07:00
				//   timePeriodWs
				//     timePeriodWsOffset
				//       00:07:31.945167 +0000
				//       00:00:00.000 +0000
				//     timePeriodWsOffsetAlpha
				//       00:07:31.945167 +0000 UTC
				//       22:18:00.001 +0000 UTC m=+0.000000001
				//       00:00:00.000 +0000 UTC
				//     timePeriodWsAlpha
				//       06:20:00.000 UTC
				if p.houri == 0 {
					p.houri = i
				}
				switch r {
				case ',':
					// hm, lets just swap out comma for period.  for some reason go
					// won't parse it.
					// 2014-05-11 08:20:13,787
					ds := []byte(p.datestr)
					ds[i] = '.'
					return ParseTime(string(ds), loc)
				case '-', '+':
					//   03:21:51+00:00
					p.stateTime = timeOffset
					if p.seci == 0 {
						// 22:18+0530
						p.Minlen = i - p.mini
					} else {
						p.Seclen = i - p.seci
					}
					p.offseti = i
				case '.':
					p.stateTime = timePeriod
					p.Seclen = i - p.seci
					p.msi = i + 1
				case 'Z':
					p.stateTime = timeZ
					if p.seci == 0 {
						p.Minlen = i - p.mini
					} else {
						p.Seclen = i - p.seci
					}
				case 'a', 'A':
					if p.nextIs(i, 't') || p.nextIs(i, 'T') {
						//                    x
						// September 17, 2012 at 5:00pm UTC-05
						i++ // skip t
						if p.nextIs(i, ' ') {
							//                      x
							// September 17, 2012 at 5:00pm UTC-05
							i++         // skip '
							p.houri = 0 // reset hour
						}
					} else {
						switch {
						case r == 'a' && p.nextIs(i, 'm'):
							p.coalesceTime(i)
							p.set(i, "am")
						case r == 'A' && p.nextIs(i, 'M'):
							p.coalesceTime(i)
							p.set(i, "PM")
						}
					}

				case 'p', 'P':
					// Could be AM/PM
					switch {
					case r == 'p' && p.nextIs(i, 'm'):
						p.coalesceTime(i)
						p.set(i, "pm")
					case r == 'P' && p.nextIs(i, 'M'):
						p.coalesceTime(i)
						p.set(i, "PM")
					}
				case ' ':
					p.coalesceTime(i)
					p.stateTime = timeWs
				case ':':
					if p.mini == 0 {
						p.mini = i + 1
						p.Hourlen = i - p.houri
					} else if p.seci == 0 {
						p.seci = i + 1
						p.Minlen = i - p.mini
					}
				}
			case timeOffset:
				// 19:55:00+0100
				// timeOffsetColon
				//   15:04:05+07:00
				//   15:04:05-07:00
				if r == ':' {
					p.stateTime = timeOffsetColon
				}
			case timeWs:
				// timeWsAlpha
				//   06:20:00 UTC
				//   06:20:00 UTC-05
				//   15:44:11 UTC+0100 2015
				//   18:04:07 GMT+0100 (GMT Daylight Time)
				//   17:57:51 MST 2009
				//   timeWsAMPMMaybe
				//     05:24:37 PM
				// timeWsOffset
				//   15:04:05 -0700
				//   00:12:00 +0000 UTC
				//   timeWsOffsetColon
				//     15:04:05 -07:00
				//     17:57:51 -0700 2009
				//     timeWsOffsetColonAlpha
				//       00:12:00 +00:00 UTC
				// timeWsYear
				//     00:12:00 2008
				// timeZ
				//   15:04:05.99Z
				switch r {
				case 'A', 'P':
					// Could be AM/PM or could be PST or similar
					p.tzi = i
					p.stateTime = timeWsAMPMMaybe
				case '+', '-':
					p.offseti = i
					p.stateTime = timeWsOffset
				default:
					if unicode.IsLetter(r) {
						// 06:20:00 UTC
						// 06:20:00 UTC-05
						// 15:44:11 UTC+0100 2015
						// 17:57:51 MST 2009
						p.tzi = i
						p.stateTime = timeWsAlpha
						//break iterTimeRunes
					} else if unicode.IsDigit(r) {
						// 00:12:00 2008
						p.stateTime = timeWsYear
						p.yeari = i
					}
				}
			case timeWsAlpha:
				// 06:20:00 UTC
				// 06:20:00 UTC-05
				// timeWsAlphaWs
				//   17:57:51 MST 2009
				// timeWsAlphaZoneOffset
				// timeWsAlphaZoneOffsetWs
				//   timeWsAlphaZoneOffsetWsExtra
				//     18:04:07 GMT+0100 (GMT Daylight Time)
				//   timeWsAlphaZoneOffsetWsYear
				//     15:44:11 UTC+0100 2015
				switch r {
				case '+', '-':
					p.tzlen = i - p.tzi
					if p.tzlen == 4 {
						p.set(p.tzi, " MST")
					} else if p.tzlen == 3 {
						p.set(p.tzi, "MST")
					}
					p.stateTime = timeWsAlphaZoneOffset
					p.offseti = i
				case ' ':
					// 17:57:51 MST 2009
					p.tzlen = i - p.tzi
					if p.tzlen == 4 {
						p.set(p.tzi, " MST")
					} else if p.tzlen == 3 {
						p.set(p.tzi, "MST")
					}
					p.stateTime = timeWsAlphaWs
					p.yeari = i + 1
				}
			case timeWsAlphaWs:
				//   17:57:51 MST 2009

			case timeWsAlphaZoneOffset:
				// 06:20:00 UTC-05
				// timeWsAlphaZoneOffset
				// timeWsAlphaZoneOffsetWs
				//   timeWsAlphaZoneOffsetWsExtra
				//     18:04:07 GMT+0100 (GMT Daylight Time)
				//   timeWsAlphaZoneOffsetWsYear
				//     15:44:11 UTC+0100 2015
				switch r {
				case ' ':
					p.set(p.offseti, "-0700")
					p.yeari = i + 1
					p.stateTime = timeWsAlphaZoneOffsetWs
				}
			case timeWsAlphaZoneOffsetWs:
				// timeWsAlphaZoneOffsetWs
				//   timeWsAlphaZoneOffsetWsExtra
				//     18:04:07 GMT+0100 (GMT Daylight Time)
				//   timeWsAlphaZoneOffsetWsYear
				//     15:44:11 UTC+0100 2015
				if unicode.IsDigit(r) {
					p.stateTime = timeWsAlphaZoneOffsetWsYear
				} else {
					p.extra = i - 1
					p.stateTime = timeWsAlphaZoneOffsetWsExtra
				}
			case timeWsAlphaZoneOffsetWsYear:
				// 15:44:11 UTC+0100 2015
				if unicode.IsDigit(r) {
					p.yearlen = i - p.yeari + 1
					if p.yearlen == 4 {
						p.setYear()
					}
				}
			case timeWsAMPMMaybe:
				// timeWsAMPMMaybe
				//   timeWsAMPM
				//     05:24:37 PM
				//   timeWsAlpha
				//     00:12:00 PST
				//     15:44:11 UTC+0100 2015
				if r == 'M' {
					//return parse("2006-01-02 03:04:05 PM", datestr, loc)
					p.stateTime = timeWsAMPM
					p.set(i-1, "PM")
					if p.Hourlen == 2 {
						p.set(p.houri, "03")
					} else if p.Hourlen == 1 {
						p.set(p.houri, "3")
					}
				} else {
					p.stateTime = timeWsAlpha
				}

			case timeWsOffset:
				// timeWsOffset
				//   15:04:05 -0700
				//   timeWsOffsetWsOffset
				//     17:57:51 -0700 -07
				//   timeWsOffsetWs
				//     17:57:51 -0700 2009
				//     00:12:00 +0000 UTC
				//   timeWsOffsetColon
				//     15:04:05 -07:00
				//     timeWsOffsetColonAlpha
				//       00:12:00 +00:00 UTC
				switch r {
				case ':':
					p.stateTime = timeWsOffsetColon
				case ' ':
					p.set(p.offseti, "-0700")
					p.yeari = i + 1
					p.stateTime = timeWsOffsetWs
				}
			case timeWsOffsetWs:
				// 17:57:51 -0700 2009
				// 00:12:00 +0000 UTC
				// 22:18:00.001 +0000 UTC m=+0.000000001
				// w Extra
				//   17:57:51 -0700 -07
				switch r {
				case '=':
					// eff you golang
					if datestr[i-1] == 'm' {
						p.extra = i - 2
						p.trimExtra()
						break
					}
				case '+', '-':
					// This really doesn't seem valid, but for some reason when round-tripping a go date
					// their is an extra +03 printed out.  seems like go bug to me, but, parsing anyway.
					// 00:00:00 +0300 +03
					// 00:00:00 +0300 +0300
					p.extra = i - 1
					p.stateTime = timeWsOffset
					p.trimExtra()
					break
				default:
					switch {
					case unicode.IsDigit(r):
						p.yearlen = i - p.yeari + 1
						if p.yearlen == 4 {
							p.setYear()
						}
					case unicode.IsLetter(r):
						if p.tzi == 0 {
							p.tzi = i
						}
					}
				}

			case timeWsOffsetColon:
				// timeWsOffsetColon
				//   15:04:05 -07:00
				//   timeWsOffsetColonAlpha
				//     2015-02-18 00:12:00 +00:00 UTC
				if unicode.IsLetter(r) {
					// 2015-02-18 00:12:00 +00:00 UTC
					p.stateTime = timeWsOffsetColonAlpha
					break iterTimeRunes
				}
			case timePeriod:
				// 15:04:05.999999999+07:00
				// 15:04:05.999999999-07:00
				// 15:04:05.999999+07:00
				// 15:04:05.999999-07:00
				// 15:04:05.999+07:00
				// 15:04:05.999-07:00
				// timePeriod
				//   17:24:37.3186369
				//   00:07:31.945167
				//   18:31:59.257000000
				//   00:00:00.000
				//   timePeriodOffset
				//     19:55:00.799+0100
				//     timePeriodOffsetColon
				//       15:04:05.999-07:00
				//   timePeriodWs
				//     timePeriodWsOffset
				//       00:07:31.945167 +0000
				//       00:00:00.000 +0000
				//       With Extra
				//         00:00:00.000 +0300 +03
				//     timePeriodWsOffsetAlpha
				//       00:07:31.945167 +0000 UTC
				//       00:00:00.000 +0000 UTC
				//       22:18:00.001 +0000 UTC m=+0.000000001
				//     timePeriodWsAlpha
				//       06:20:00.000 UTC
				switch r {
				case ' ':
					p.Mslen = i - p.msi
					p.stateTime = timePeriodWs
				case '+', '-':
					// This really shouldn't happen
					p.Mslen = i - p.msi
					p.offseti = i
					p.stateTime = timePeriodOffset
				default:
					if unicode.IsLetter(r) {
						// 06:20:00.000 UTC
						p.Mslen = i - p.msi
						p.stateTime = timePeriodWsAlpha
					}
				}
			case timePeriodOffset:
				// timePeriodOffset
				//   19:55:00.799+0100
				//   timePeriodOffsetColon
				//     15:04:05.999-07:00
				//     13:31:51.999-07:00 MST
				if r == ':' {
					p.stateTime = timePeriodOffsetColon
				}
			case timePeriodOffsetColon:
				// timePeriodOffset
				//   timePeriodOffsetColon
				//     15:04:05.999-07:00
				//     13:31:51.999 -07:00 MST
				switch r {
				case ' ':
					p.set(p.offseti, "-07:00")
					p.stateTime = timePeriodOffsetColonWs
					p.tzi = i + 1
				}
			case timePeriodOffsetColonWs:
				// continue
			case timePeriodWs:
				// timePeriodWs
				//   timePeriodWsOffset
				//     00:07:31.945167 +0000
				//     00:00:00.000 +0000
				//   timePeriodWsOffsetAlpha
				//     00:07:31.945167 +0000 UTC
				//     00:00:00.000 +0000 UTC
				//   timePeriodWsOffsetColon
				//     13:31:51.999 -07:00 MST
				//   timePeriodWsAlpha
				//     06:20:00.000 UTC
				if p.offseti == 0 {
					p.offseti = i
				}
				switch r {
				case '+', '-':
					p.Mslen = i - p.msi - 1
					p.stateTime = timePeriodWsOffset
				default:
					if unicode.IsLetter(r) {
						//     00:07:31.945167 +0000 UTC
						//     00:00:00.000 +0000 UTC
						p.stateTime = timePeriodWsOffsetWsAlpha
						break iterTimeRunes
					}
				}

			case timePeriodWsOffset:
				// timePeriodWs
				//   timePeriodWsOffset
				//     00:07:31.945167 +0000
				//     00:00:00.000 +0000
				//     With Extra
				//       00:00:00.000 +0300 +03
				//   timePeriodWsOffsetAlpha
				//     00:07:31.945167 +0000 UTC
				//     00:00:00.000 +0000 UTC
				//     03:02:00.001 +0300 MSK m=+0.000000001
				//   timePeriodWsOffsetColon
				//     13:31:51.999 -07:00 MST
				//   timePeriodWsAlpha
				//     06:20:00.000 UTC
				switch r {
				case ':':
					p.stateTime = timePeriodWsOffsetColon
				case ' ':
					p.set(p.offseti, "-0700")
				case '+', '-':
					// This really doesn't seem valid, but for some reason when round-tripping a go date
					// their is an extra +03 printed out.  seems like go bug to me, but, parsing anyway.
					// 00:00:00.000 +0300 +03
					// 00:00:00.000 +0300 +0300
					p.extra = i - 1
					p.trimExtra()
					break
				default:
					if unicode.IsLetter(r) {
						// 00:07:31.945167 +0000 UTC
						// 00:00:00.000 +0000 UTC
						// 03:02:00.001 +0300 MSK m=+0.000000001
						p.stateTime = timePeriodWsOffsetWsAlpha
					}
				}
			case timePeriodWsOffsetWsAlpha:
				// 03:02:00.001 +0300 MSK m=+0.000000001
				// eff you golang
				if r == '=' && datestr[i-1] == 'm' {
					p.extra = i - 2
					p.trimExtra()
					break
				}

			case timePeriodWsOffsetColon:
				// 13:31:51.999 -07:00 MST
				switch r {
				case ' ':
					p.set(p.offseti, "-07:00")
				default:
					if unicode.IsLetter(r) {
						// 13:31:51.999 -07:00 MST
						p.tzi = i
						p.stateTime = timePeriodWsOffsetColonAlpha
					}
				}
			case timePeriodWsOffsetColonAlpha:
				// continue
			case timeZ:
				// timeZ
				//   15:04:05.99Z
				// With a time-zone at end after Z
				// 2006-01-02T15:04:05.999999999Z07:00
				// 2006-01-02T15:04:05Z07:00
				// RFC3339     = "2006-01-02T15:04:05Z07:00"
				// RFC3339Nano = "2006-01-02T15:04:05.999999999Z07:00"
				if unicode.IsDigit(r) {
					p.stateTime = timeZDigit
				}

			}
		}

		switch p.stateTime {
		case timeWsAlphaWs:
			p.yearlen = i - p.yeari
			p.setYear()
		case timeWsYear:
			p.yearlen = i - p.yeari
			p.setYear()
		case timeWsAlphaZoneOffsetWsExtra:
			p.trimExtra()
		case timeWsAlphaZoneOffset:
			// 06:20:00 UTC-05
			if i-p.offseti < 4 {
				p.set(p.offseti, "-07")
			} else {
				p.set(p.offseti, "-0700")
			}

		case timePeriod:
			p.Mslen = i - p.msi
		case timeOffset:
			// 19:55:00+0100
			p.set(p.offseti, "-0700")
		case timeWsOffset:
			p.set(p.offseti, "-0700")
		case timeWsOffsetWs:
			// 17:57:51 -0700 2009
			// 00:12:00 +0000 UTC
		case timeWsOffsetColon:
			// 17:57:51 -07:00
			p.set(p.offseti, "-07:00")
		case timeOffsetColon:
			// 15:04:05+07:00
			p.set(p.offseti, "-07:00")
		case timePeriodOffset:
			// 19:55:00.799+0100
			p.set(p.offseti, "-0700")
		case timePeriodOffsetColon:
			p.set(p.offseti, "-07:00")
		case timePeriodWsOffsetColonAlpha:
			p.tzlen = i - p.tzi
			switch p.tzlen {
			case 3:
				p.set(p.tzi, "MST")
			case 4:
				p.set(p.tzi, "MST ")
			}
		case timePeriodWsOffset:
			p.set(p.offseti, "-0700")
		}
		p.coalesceTime(i)
	}

	switch p.stateDate {
	case dateDigit:
		// unixy timestamps ish
		//  example              ct type
		//  1499979655583057426  19 nanoseconds
		//  1499979795437000     16 micro-seconds
		//  20180722105203       14 yyyyMMddhhmmss
		//  1499979795437        13 milliseconds
		//  1332151919           10 seconds
		//  20140601             8  yyyymmdd
		//  2014                 4  yyyy
		t := time.Time{}
		if len(datestr) == len("1499979655583057426") { // 19
			// nano-seconds
			if nanoSecs, err := strconv.ParseInt(datestr, 10, 64); err == nil {
				t = time.Unix(0, nanoSecs)
			}
		} else if len(datestr) == len("1499979795437000") { // 16
			// micro-seconds
			if microSecs, err := strconv.ParseInt(datestr, 10, 64); err == nil {
				t = time.Unix(0, microSecs*1000)
			}
		} else if len(datestr) == len("yyyyMMddhhmmss") { // 14
			// yyyyMMddhhmmss
			p.Format = []byte("20060102150405")
			return p, nil
		} else if len(datestr) == len("1332151919000") { // 13
			if miliSecs, err := strconv.ParseInt(datestr, 10, 64); err == nil {
				t = time.Unix(0, miliSecs*1000*1000)
			}
		} else if len(datestr) == len("1332151919") { //10
			if secs, err := strconv.ParseInt(datestr, 10, 64); err == nil {
				t = time.Unix(secs, 0)
			}
		} else if len(datestr) == len("20140601") {
			p.Format = []byte("20060102")
			return p, nil
		} else if len(datestr) == len("2014") {
			p.Format = []byte("2006")
			return p, nil
		} else if len(datestr) < 4 {
			return nil, fmt.Errorf("unrecognized format, too short %v", datestr)
		}
		if !t.IsZero() {
			if loc == nil {
				p.t = &t
				return p, nil
			}
			t = t.In(loc)
			p.t = &t
			return p, nil
		}

	case dateYearDash:
		// 2006-01
		return p, nil

	case dateYearDashDash:
		// 2006-01-02
		// 2006-1-02
		// 2006-1-2
		// 2006-01-2
		return p, nil

	case dateYearDashAlphaDash:
		// 2013-Feb-03
		// 2013-Feb-3
		p.daylen = i - p.dayi
		p.setDay()
		return p, nil

	case dateYearDashDashWs:
		// 2013-04-01
		return p, nil

	case dateYearDashDashT:
		return p, nil

	case dateDigitDashAlphaDash:
		// 13-Feb-03   ambiguous
		// 28-Feb-03   ambiguous
		// 29-Jun-2016
		return p, nil

	case dateDigitDot:
		// 2014.05
		p.molen = i - p.moi
		p.setMonth()
		return p, nil

	case dateDigitDotDot:
		// 03.31.1981
		// 3.31.2014
		// 3.2.1981
		// 3.2.81
		// 08.21.71
		// 2018.09.30
		return p, nil

	case dateDigitWsMoYear:
		// 2 Jan 2018
		// 2 Jan 18
		// 2 Jan 2018 23:59
		// 02 Jan 2018 23:59
		// 12 Feb 2006, 19:17
		return p, nil

	case dateDigitWsMolong:
		// 18 January 2018
		// 8 January 2018
		if p.daylen == 2 {
			p.Format = []byte("02 January 2006")
			return p, nil
		}
		p.Format = []byte("2 January 2006")
		return p, nil // parse("2 January 2006", datestr, loc)

	case dateAlphaWsMonth:
		p.yearlen = i - p.yeari
		p.setYear()
		return p, nil

	case dateAlphaWsMonthMore:
		return p, nil

	case dateAlphaWsDigitMoreWs:
		// oct 1, 1970
		p.yearlen = i - p.yeari
		p.setYear()
		return p, nil

	case dateAlphaWsDigitMoreWsYear:
		// May 8, 2009 5:57:51 PM
		// Jun 7, 2005, 05:57:51
		return p, nil

	case dateAlphaWsAlpha:
		return p, nil

	case dateAlphaWsAlphaYearmaybe:
		return p, nil

	case dateDigitSlash:
		// 3/1/2014
		// 10/13/2014
		// 01/02/2006
		// 2014/10/13
		return p, nil

	case dateDigitChineseYear:
		// dateDigitChineseYear
		//   2014年04月08日
		p.Format = []byte("2006年01月02日")
		return p, nil

	case dateDigitChineseYearWs:
		p.Format = []byte("2006年01月02日 15:04:05")
		return p, nil

	case dateWeekdayComma:
		// Monday, 02 Jan 2006 15:04:05 -0700
		// Monday, 02 Jan 2006 15:04:05 +0100
		// Monday, 02-Jan-06 15:04:05 MST
		return p, nil

	case dateWeekdayAbbrevComma:
		// Mon, 02-Jan-06 15:04:05 MST
		// Mon, 02 Jan 2006 15:04:05 MST
		return p, nil

	}

	return nil, unknownErr(datestr)
}

type parser struct {
	loc              *time.Location
	preferMonthFirst bool
	ambiguousMD      bool
	stateDate        dateState
	stateTime        timeState
	Format           []byte
	datestr          string
	fullMonth        string
	skip             int
	extra            int
	part1Len         int
	yeari            int
	yearlen          int
	moi              int
	molen            int
	dayi             int
	daylen           int
	houri            int
	Hourlen          int
	mini             int
	Minlen           int
	seci             int
	Seclen           int
	msi              int
	Mslen            int
	offseti          int
	offsetlen        int
	tzi              int
	tzlen            int
	t                *time.Time
}

func newParser(dateStr string, loc *time.Location) *parser {
	p := parser{
		stateDate:        dateStart,
		stateTime:        timeIgnore,
		datestr:          dateStr,
		loc:              loc,
		preferMonthFirst: true,
	}
	p.Format = []byte(dateStr)
	return &p
}

func (p *parser) nextIs(i int, b byte) bool {
	if len(p.datestr) > i+1 && p.datestr[i+1] == b {
		return true
	}
	return false
}

func (p *parser) set(start int, val string) {
	if start < 0 {
		return
	}
	if len(p.Format) < start+len(val) {
		return
	}
	for i, r := range val {
		p.Format[start+i] = byte(r)
	}
}
func (p *parser) setMonth() {
	if p.molen == 2 {
		p.set(p.moi, "01")
	} else if p.molen == 1 {
		p.set(p.moi, "1")
	}
}

func (p *parser) setDay() {
	if p.daylen == 2 {
		p.set(p.dayi, "02")
	} else if p.daylen == 1 {
		p.set(p.dayi, "2")
	}
}
func (p *parser) setYear() {
	if p.yearlen == 2 {
		p.set(p.yeari, "06")
	} else if p.yearlen == 4 {
		p.set(p.yeari, "2006")
	}
}
func (p *parser) coalesceDate(end int) {
	if p.yeari > 0 {
		if p.yearlen == 0 {
			p.yearlen = end - p.yeari
		}
		p.setYear()
	}
	if p.moi > 0 && p.molen == 0 {
		p.molen = end - p.moi
		p.setMonth()
	}
	if p.dayi > 0 && p.daylen == 0 {
		p.daylen = end - p.dayi
		p.setDay()
	}
}
func (p *parser) ts() string {
	return fmt.Sprintf("h:(%d:%d) m:(%d:%d) s:(%d:%d)", p.houri, p.Hourlen, p.mini, p.Minlen, p.seci, p.Seclen)
}
func (p *parser) ds() string {
	return fmt.Sprintf("%s d:(%d:%d) m:(%d:%d) y:(%d:%d)", p.datestr, p.dayi, p.daylen, p.moi, p.molen, p.yeari, p.yearlen)
}
func (p *parser) coalesceTime(end int) {
	// 03:04:05
	// 15:04:05
	// 3:04:05
	// 3:4:5
	// 15:04:05.00
	if p.houri > 0 {
		if p.Hourlen == 2 {
			p.set(p.houri, "15")
		} else if p.Hourlen == 1 {
			p.set(p.houri, "3")
		}
	}
	if p.mini > 0 {
		if p.Minlen == 0 {
			p.Minlen = end - p.mini
		}
		if p.Minlen == 2 {
			p.set(p.mini, "04")
		} else {
			p.set(p.mini, "4")
		}
	}
	if p.seci > 0 {
		if p.Seclen == 0 {
			p.Seclen = end - p.seci
		}
		if p.Seclen == 2 {
			p.set(p.seci, "05")
		} else {
			p.set(p.seci, "5")
		}
	}

	if p.msi > 0 {
		for i := 0; i < p.Mslen; i++ {
			p.Format[p.msi+i] = '0'
		}
	}
}
func (p *parser) setFullMonth(month string) {
	if p.moi == 0 {
		p.Format = []byte(fmt.Sprintf("%s%s", "January", p.Format[len(month):]))
	}
}

func (p *parser) trimExtra() {
	if p.extra > 0 && len(p.Format) > p.extra {
		p.Format = p.Format[0:p.extra]
		p.datestr = p.datestr[0:p.extra]
	}
}

// func (p *parser) remove(i, length int) {
// 	if len(p.Format) > i+length {
// 		//append(a[:i], a[j:]...)
// 		p.Format = append(p.Format[0:i], p.Format[i+length:]...)
// 	}
// 	if len(p.datestr) > i+length {
// 		//append(a[:i], a[j:]...)
// 		p.datestr = fmt.Sprintf("%s%s", p.datestr[0:i], p.datestr[i+length:])
// 	}
// }

func (p *parser) Parse() (time.Time, error) {
	if p.t != nil {
		return *p.t, nil
	}
	if len(p.fullMonth) > 0 {
		p.setFullMonth(p.fullMonth)
	}
	if p.skip > 0 && len(p.Format) > p.skip {
		p.Format = p.Format[p.skip:]
		p.datestr = p.datestr[p.skip:]
	}
	//gou.Debugf("parse %q   AS   %q", p.datestr, string(p.Format))
	if p.loc == nil {
		return time.Parse(string(p.Format), p.datestr)
	}
	return time.ParseInLocation(string(p.Format), p.datestr, p.loc)
}
func isMonthFull(alpha string) bool {
	for _, month := range months {
		if alpha == month {
			return true
		}
	}
	return false
}
