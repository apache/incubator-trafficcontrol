package tc

/*
 * Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements.  See the NOTICE file
 * distributed with this work for additional information
 * regarding copyright ownership.  The ASF licenses this file
 * to you under the Apache License, Version 2.0 (the
 * "License"); you may not use this file except in compliance
 * with the License.  You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

import (
	"errors"
	"fmt"
	"reflect"
	"testing"
)

func ExampleCreateErrorAlerts() {
	alerts := CreateErrorAlerts(errors.New("foo"))
	fmt.Printf("%v\n", alerts)
	// Output: {[{foo error}]}
}

func ExampleCreateAlerts() {
	alerts := CreateAlerts(InfoLevel, "foo", "bar")
	fmt.Printf("%d\n", len(alerts.Alerts))
	fmt.Printf("Level: %s, Text: %s\n", alerts.Alerts[0].Level, alerts.Alerts[0].Text)
	fmt.Printf("Level: %s, Text: %s\n", alerts.Alerts[1].Level, alerts.Alerts[1].Text)

	// Output: 2
	// Level: info, Text: foo
	// Level: info, Text: bar
}

func ExampleAlerts_ToStrings() {
	alerts := CreateAlerts(InfoLevel, "foo", "bar")
	strs := alerts.ToStrings()
	fmt.Printf("%d\n%s\n%s\n", len(strs), strs[0], strs[1])
	// Output: 2
	// foo
	// bar
}

func ExampleAlerts_AddNewAlert() {
	var alerts Alerts
	fmt.Printf("%d\n", len(alerts.Alerts))
	alerts.AddNewAlert(InfoLevel, "foo")
	fmt.Printf("%d\n", len(alerts.Alerts))
	fmt.Printf("Level: %s, Text: %s\n", alerts.Alerts[0].Level, alerts.Alerts[0].Text)

	// Output: 0
	// 1
	// Level: info, Text: foo
}

func ExampleAlerts_AddAlert() {
	var alerts Alerts
	fmt.Printf("%d\n", len(alerts.Alerts))
	alert := Alert{
		Level: InfoLevel.String(),
		Text:  "foo",
	}
	alerts.AddAlert(alert)
	fmt.Printf("%d\n", len(alerts.Alerts))
	fmt.Printf("Level: %s, Text: %s\n", alerts.Alerts[0].Level, alerts.Alerts[0].Text)

	// Output: 0
	// 1
	// Level: info, Text: foo
}

func ExampleAlerts_AddAlerts() {
	alerts1 := Alerts{
		[]Alert{
			{
				Level: InfoLevel.String(),
				Text:  "foo",
			},
		},
	}
	alerts2 := Alerts{
		[]Alert{
			{
				Level: ErrorLevel.String(),
				Text:  "bar",
			},
		},
	}

	alerts1.AddAlerts(alerts2)
	fmt.Printf("%d\n", len(alerts1.Alerts))
	fmt.Printf("Level: %s, Text: %s\n", alerts1.Alerts[0].Level, alerts1.Alerts[0].Text)
	fmt.Printf("Level: %s, Text: %s\n", alerts1.Alerts[1].Level, alerts1.Alerts[1].Text)

	// Output: 2
	// Level: info, Text: foo
	// Level: error, Text: bar
}

func ExampleAlerts_ErrorString() {
	alerts := CreateErrorAlerts(errors.New("foo"), errors.New("bar"))
	fmt.Println(alerts.ErrorString())

	alerts = CreateAlerts(WarnLevel, "test")
	alerts.AddAlert(Alert{Level: InfoLevel.String(), Text: "quest"})
	fmt.Println(alerts.ErrorString())

	// Output: foo; bar
	//
}

func TestCreateAlerts(t *testing.T) {
	expected := Alerts{[]Alert{}}
	alerts := CreateAlerts(WarnLevel)
	if !reflect.DeepEqual(expected, alerts) {
		t.Errorf("Expected %v Got %v", expected, alerts)
	}

	expected = Alerts{[]Alert{{"message 1", WarnLevel.String()}, {"message 2", WarnLevel.String()}, {"message 3", WarnLevel.String()}}}
	alerts = CreateAlerts(WarnLevel, "message 1", "message 2", "message 3")
	if !reflect.DeepEqual(expected, alerts) {
		t.Errorf("Expected %v Got %v", expected, alerts)
	}
}
