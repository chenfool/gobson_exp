// Copyright 2015-2016 David Li
//
// Licensed under the Apache License, Version 2.0 (the "License"): you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

package bson

import "testing"

func TestByteOrder(t *testing.T) {
	x := int32(0x0AEBEC0D)
	y := int32(0x0DECEB0A)
	r := RevertInt32(x)
	if r != y {
		t.Errorf("expect %v, get %v", y, r)
	}
}
