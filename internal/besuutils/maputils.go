// Copyright 2018, 2019 SEE CONTRIBUTORS

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

//     http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package besudutils

import "reflect"

// GetMapString is a helper to safely extract strings from generic interface maps
func GetMapString(genericMap map[string]interface{}, key string) string {
	if val, exists := genericMap[key]; exists {
		if reflect.TypeOf(val).Kind() == reflect.String {
			return val.(string)
		}
	}
	return ""
}
