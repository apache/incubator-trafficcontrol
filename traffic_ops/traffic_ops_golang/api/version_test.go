package api

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

import "fmt"

func ExampleVersion_String() {
	v := Version{
		Major: 1,
		Minor: 2,
	}
	fmt.Println(v)

	// Output: 1.2
}

func ExampleVersion_Equal() {
	v := Version{
		Major: 1,
		Minor: 2,
	}
	o := Version{
		Major: 1,
		Minor: 2,
	}
	fmt.Println(v.Equal(o))
	fmt.Println(o.Equal(v))

	o.Major++
	fmt.Println(v.Equal(o))
	fmt.Println(o.Equal(v))

	o.Major--
	o.Minor--
	fmt.Println(v.Equal(o))
	fmt.Println(o.Equal(v))

	// Output: true
	// true
	// false
	// false
	// false
	// false
}

func ExampleVersion_GreaterThan() {
	v := Version{
		Major: 1,
		Minor: 2,
	}
	o := Version{
		Major: 1,
		Minor: 2,
	}

	fmt.Println(v.GreaterThan(o))
	fmt.Println(o.GreaterThan(v))

	o.Major--
	fmt.Println(v.GreaterThan(o))
	fmt.Println(o.GreaterThan(v))

	o.Major++
	o.Minor--
	fmt.Println(v.GreaterThan(o))
	fmt.Println(o.GreaterThan(v))

	// Output: false
	// false
	// true
	// false
	// true
	// false
}

func ExampleVersion_LessThan() {
	v := Version{
		Major: 1,
		Minor: 2,
	}
	o := Version{
		Major: 1,
		Minor: 2,
	}

	fmt.Println(v.LessThan(o))
	fmt.Println(o.LessThan(v))

	o.Major--
	fmt.Println(v.LessThan(o))
	fmt.Println(o.LessThan(v))

	o.Major++
	o.Minor--
	fmt.Println(v.LessThan(o))
	fmt.Println(o.LessThan(v))

	// Output: false
	// false
	// false
	// true
	// false
	// true
}
