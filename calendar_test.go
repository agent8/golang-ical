package ics

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
)

func TestCalendarStream(t *testing.T) {
	i := `
ATTENDEE;RSVP=TRUE;ROLE=REQ-PARTICIPANT;CUTYPE=GROUP:
 mailto:employee-A@example.com
DESCRIPTION:Project XYZ Review Meeting
CATEGORIES:MEETING
CLASS:PUBLIC
`
	expected := []ContentLine{
		ContentLine("ATTENDEE;RSVP=TRUE;ROLE=REQ-PARTICIPANT;CUTYPE=GROUP:mailto:employee-A@example.com"),
		ContentLine("DESCRIPTION:Project XYZ Review Meeting"),
		ContentLine("CATEGORIES:MEETING"),
		ContentLine("CLASS:PUBLIC"),
	}
	c := NewCalendarStream(strings.NewReader(i))
	cont := true
	for i := 0; cont; i++ {
		l, err := c.ReadLine()
		if err != nil {
			switch err {
			case io.EOF:
				cont = false
			default:
				t.Logf("Unknown error; %v", err)
				t.Fail()
				return
			}
		}
		if l == nil {
			if err == io.EOF && i == len(expected) {
				cont = false
			} else {
				t.Logf("Nil response...")
				t.Fail()
				return
			}
		}
		if i < len(expected) {
			if string(*l) != string(expected[i]) {
				t.Logf("Got %s expected %s", string(*l), string(expected[i]))
				t.Fail()
			}
		} else if l != nil {
			t.Logf("Larger than expected")
			t.Fail()
			return
		}
	}
}

func TestRfc5545Sec4Examples(t *testing.T) {
	rnReplace := regexp.MustCompile("\r?\n")

	err := filepath.Walk("./testdata/rfc5545sec4/", func(path string, info os.FileInfo, _ error) error {
		if info.IsDir() {
			return nil
		}

		inputBytes, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}

		input := rnReplace.ReplaceAllString(string(inputBytes), "\r\n")
		structure, err := ParseCalendar(strings.NewReader(input))
		if assert.Nil(t, err, path) {
			// This should fail as the sample data doesn't conform to https://tools.ietf.org/html/rfc5545#page-45
			// Probably due to RFC width guides
			assert.NotNil(t, structure)

			output := structure.Serialize()
			assert.NotEqual(t, input, output)
		}

		return nil
	})

	if err != nil {
		t.Fatalf("cannot read test directory: %v", err)
	}
}

func TestLineFolding(t *testing.T) {
	testCases := []struct {
		name   string
		input  string
		output string
	}{
		{
			name:  "fold lines at nearest space",
			input: "some really long line with spaces to fold on and the line should fold",
			output: `BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//arran4//Golang ICS Library
DESCRIPTION:some really long line with spaces to fold on and the line
  should fold
END:VCALENDAR
`,
		},
		{
			name:  "fold lines if no space",
			input: "somereallylonglinewithnospacestofoldonandthelineshouldfoldtothenextline",
			output: `BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//arran4//Golang ICS Library
DESCRIPTION:somereallylonglinewithnospacestofoldonandthelineshouldfoldtothe
 nextline
END:VCALENDAR
`,
		},
		{
			name:  "fold lines at nearest space",
			input: "some really long line with spaces howeverthelastpartofthelineisactuallytoolongtofitonsowehavetofoldpartwaythrough",
			output: `BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//arran4//Golang ICS Library
DESCRIPTION:some really long line with spaces
  howeverthelastpartofthelineisactuallytoolongtofitonsowehavetofoldpartwayt
 hrough
END:VCALENDAR
`,
		},
		{
			name:  "75 chars line should not fold",
			input: " this line is exactly 75 characters long with the property name",
			output: `BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//arran4//Golang ICS Library
DESCRIPTION: this line is exactly 75 characters long with the property name
END:VCALENDAR
`,
		},
		{
			name: "runes should not be split",
			// the 75 bytes mark is in the middle of a rune
			input: "éé界世界世界世界世界世界世界世界世界世界世界世界世界",
			output: `BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//arran4//Golang ICS Library
DESCRIPTION:éé界世界世界世界世界世界世界世界世界世界
 世界世界世界
END:VCALENDAR
`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			c := NewCalendar()
			c.SetDescription(tc.input)
			// we're not testing for encoding here so lets make the actual output line breaks == expected line breaks
			text := strings.Replace(c.Serialize(), "\r\n", "\n", -1)

			assert.Equal(t, tc.output, text)
			assert.True(t, utf8.ValidString(text), "Serialized .ics calendar isn't valid UTF-8 string")
		})
	}
}

func TestTimeZone(t *testing.T) {
	data1 := `
BEGIN:VCALENDAR
PRODID:-//Google Inc//Google Calendar 70.9054//EN
VERSION:2.0
CALSCALE:GREGORIAN
METHOD:REQUEST

BEGIN:VTIMEZONE
TZID:Taipei Standard Time
TZURL:http://timezones.example.org/tz/America-Los_Angeles.ics

BEGIN:STANDARD
DTSTART:16010101T000000
TZOFFSETFROM:+0800
TZOFFSETTO:+0800
RRULE:FREQ=YEARLY;BYMONTH=4;BYDAY=-1SU;UNTIL=19730429T070000Z
TZNAME:EDT
END:STANDARD

BEGIN:DAYLIGHT
DTSTART:16010101T000000
TZOFFSETFROM:+0800
TZOFFSETTO:+0800
RRULE:FREQ=YEARLY;BYMONTH=10;BYDAY=-1SU;UNTIL=20061029T060000Z
TZNAME:EST
END:DAYLIGHT

END:VTIMEZONE



BEGIN:VTIMEZONE
TZID:America/New_York
LAST-MODIFIED:20050809T050000Z
BEGIN:STANDARD
DTSTART:20071104T020000
TZOFFSETFROM:-0400
TZOFFSETTO:-0500
TZNAME:EST
END:STANDARD
BEGIN:DAYLIGHT
DTSTART:20070311T020000
TZOFFSETFROM:-0500
TZOFFSETTO:-0400
TZNAME:EDT
END:DAYLIGHT
END:VTIMEZONE

BEGIN:VEVENT
DTSTART;tzid=Taipei Standard Time:20211112T000000
DTEND;TZID=Taipei Standard Time:20211112T010000
ORGANIZER;CN=bonnie@edison.tech:mailto:bonnie@edison.tech
UID:69b37lqafm98nr7jvu56f8utv8@google.com
ATTENDEE;CUTYPE=INDIVIDUAL;ROLE=REQ-PARTICIPANT;PARTSTAT=ACCEPTED;RSVP=TRUE
 ;CN=myan@yipitdata.com;X-NUM-GUESTS=0:mailto:myan@yipitdata.com
CREATED:20211025T162806Z
DESCRIPTION:Here is the video and audio links.
LAST-MODIFIED:20211026T152153Z
LOCATION:Zoom info in details // link for English and Tencent link for simu
 ltaneous Chinese translation.( Audio only)
SEQUENCE:0
STATUS:CONFIRMED
SUMMARY:Vin x Edison China Team town hall
TRANSP:OPAQUE
END:VEVENT

BEGIN:VEVENT
UID:19970901T130000Z-123403@example.com
DTSTAMP:19970901T130000Z
DTSTART;VALUE=DATE:19971102
SUMMARY:Our Blissful Anniversary
TRANSP:TRANSPARENT
CLASS:CONFIDENTIAL
CATEGORIES:ANNIVERSARY,PERSONAL,SPECIAL OCCASION
RRULE:FREQ=YEARLY
END:VEVENT

END:VCALENDAR
`

	calendar, _ := ParseCalendar(strings.NewReader(data1))
	t.Log("calendar:", calendar)
	t.Log("-------------------")

	for _, timezone := range calendar.Timezones() {
		t.Log("timezone:", timezone)
		t.Log("timezone.tzid", timezone.GetId())
		t.Log("timezone.tzurl", timezone.GetUrl())

		for _, standard := range timezone.GetStands() {
			t.Log("standard:", standard)

			t.Log("DTSTART:", standard.GetDtStart())
			t.Log("TZOFFSETFROM:", standard.GetTzOffsetFrom())
			t.Log("TZOFFSETTO:", standard.GetTzOffsetTo())
			t.Log("TZNAME:", standard.GetTzName())
			t.Log("RRULE:", standard.GetRRule())
			t.Log("RDATE:", standard.GetRDate())

			t.Log("-------------------")
		}

		for _, daylight := range timezone.GetDaylights() {
			t.Log("daylight:", daylight)

			t.Log("DTSTART:", daylight.GetDtStart())
			t.Log("TZOFFSETFROM:", daylight.GetTzOffsetFrom())
			t.Log("TZOFFSETTO:", daylight.GetTzOffsetTo())
			t.Log("TZNAME:", daylight.GetTzName())
			t.Log("RRULE:", daylight.GetRRule())
			t.Log("RDATE:", daylight.GetRDate())

			t.Log("-------------------")
		}

		for _, observance := range timezone.GetAllObservances() {
			t.Log("observance:", observance, "Type", observance.Type)

			t.Log("DTSTART:", observance.GetDtStart())
			t.Log("TZOFFSETFROM:", observance.GetTzOffsetFrom())
			t.Log("TZOFFSETTO:", observance.GetTzOffsetTo())
			t.Log("TZNAME:", observance.GetTzName())
			t.Log("RRULE:", observance.GetRRule())
			t.Log("RDATE:", observance.GetRDate())
		}

		t.Log("-------------------")
	}

	timezone := calendar.FindTimezone("America/New_York")
	t.Log("timezone: \"America/New_York\" :", timezone)
	t.Log("timezone.tzid:", timezone.GetId())
	t.Log("timezone.tzurl:", timezone.GetUrl())

	t.Log("-------------------")

	for _, event := range calendar.Events() {
		t.Log("event:", event)

		pro := event.GetProperty(ComponentProperty(PropertyDtstart))
		t.Log("PropertyDtstart:", pro)

		tzid := pro.ICalParameters[string(ParameterTzid)]
		if len(tzid) > 0 {
			t.Log("tzid:", tzid[0])
		}

		value := pro.ICalParameters[string(ParameterValue)]
		if len(value) > 0 {
			t.Log("date-time type:", value[0])
		}

		s := pro.Value
		t.Log("stamp:", s)
		t.Log("-------------------")
	}

	t.Log("-------------------")
}
