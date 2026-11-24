/*
 *
 *    Copyright 2026 Han Li and contributors
 *
 *    Licensed under the Apache License, Version 2.0 (the "License");
 *    you may not use this file except in compliance with the License.
 *    You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 *    Unless required by applicable law or agreed to in writing, software
 *    distributed under the License is distributed on an "AS IS" BASIS,
 *    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *    See the License for the specific language governing permissions and
 *    limitations under the License.
 *
 */

package sdk

import "errors"

var ErrRuntimeNotFound = errors.New("runtime not found")

// ErrFilesLeft is returned by Uninstall when the version is no longer installed,
// but some of its files could not be deleted.
var ErrFilesLeft = errors.New("could not delete all files of")

// ErrLinksLeft is returned by Uninstall when the version is no longer installed,
// but some of the global links into it could not be removed.
var ErrLinksLeft = errors.New("could not remove all links to")
