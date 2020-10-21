// Copyright © Microsoft <wastore@microsoft.com>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package e2etest

import (
	"testing"
	"time"
)

// Purpose: Tests for the filtering functionality (when enumerating sources)

// Please leave the following test at the top of this file, where it can serve as an easy-to-find annotated example.
// We won't normally put this many comments in a test, but this one has the verbose comments to explain the declarative test
// framework
func TestFilter_IncludePath(t *testing.T) {
	// This will test IncludePath once for each source resource type.
	// For source resource types that support both Copy and Sync, it will run the test twice, once with Copy and once with Sync.
	//  Copy: Blob -> Blob
	//  Copy: Local -> Blob
	//  Copy: Files -> Files
	//  Copy: AWS -> Blob
	//  Copy: ADLS Gen2 -> Local
	// That's 5 scenarios in total, but we only need to specify the test declaratively _once_.  The eOperation and eTestFromTo
	// parameters automatically cause this test to expand out to the 5 scenarios. (If we had specified eOperation.CopyAndSync()
	// instead of just eOperation.Copy(), then for the first three listed above, RunTests would have run Sync as well, making
	// it 8 scenarios in total. But include-path does not apply to Sync, so we did not specify that here)

	RunScenarios( // This is the method that does all the work.  We pass it params to define that test that should be run
		t,                                 // Pass in the test context
		eOperation.Copy(),                 // Should the test be run for copy only, sync only, or both?
		eTestFromTo.AllSourcesToOneDest(), // What range of source/dest pairs should this test be run on
		eValidate.Auto(),                  // What to validate (in this case, we don't validate content. We just validate that the desired transfers were scheduled
		params{ // Pass flag values that the test requires. The params struct is a superset of Copy and Sync params
			recursive:   true,
			includePath: "sub/subsub;wantedfile",
		},
		nil, // For advanced usage, can pass a hooks struct here, to hook funcs into different stage of the testing process to customize it
		testFiles{ // Source files specifies details of the files to test on
			defaultSize: "1K", // An indication of what size of files should be created
			shouldIgnore: []interface{}{ // A list of files which should be created, but which are expected to be ignored by the job
				folder(""), // root folder (i.e. the folder that normally gets copied when source doesn't end in /*.  But it doesn't get copied in this case, because it doesn't match the include-path)
				"filea",
				"fileb",
				"filec",
				"wantedfileabc", // include-path only works with whole filenames, so this won't match wantedfile
				"sub/filea",
				"sub/fileb",
				"sub/filec",
				folder("sub/subsubsub"),          // include-path only works with _whole_ directories (i.e. not prefix match)
				"sub/somethingelse/subsub/filey", // should not be included because sub/subsub is not contiguous here
				"othersub/sub/subsub/filey",      // should not be included because sub/subsub is not at root here
				"othersub/wantedfile",            // should not be included because, although wantedfile is in the includepath, include path always starts from the root
			},
			shouldTransfer: []interface{}{ // A list of files which should be created an which should indeed be transferred
				// Include folders as a line that ends in /. Test framework will automatically ignore them when
				// not transferring between folder-aware locations
				"wantedfile",
				folder("sub/subsub"),
				"sub/subsub/filea",
				"sub/subsub/fileb",
				"sub/subsub/filec",
			},
		},
	)
}

// TestFilter_IncludeAfter test the include-after parameter
func TestFilter_IncludeAfter(t *testing.T) {
	RunScenarios(
		t,
		eOperation.Copy(), // IncludeAfter is not applicable for sync
		eTestFromTo.AllSourcesToOneDest(),
		eValidate.Auto(),
		params{
			recursive: true,
		},
		&hooks{
			beforeRunJob: func(h hookHelper) {
				// let LMTs of existing file age a little (so they are definitely older than our include-after)
				time.Sleep(4 * time.Second)

				// set includeAfter to "now"
				scenarioParams := h.GetModifiableParameters()
				scenarioParams.includeAfter = time.Now().Format(time.RFC3339)

				// wait a moment, so that LMTs of the files we are about to create will be definitely >= our include-after
				// (without this, we had a bug, presumably due to a small clock skew error between client machine and blob storage,
				// in which the LMTs of the re-created files ended up before the include-after time).
				time.Sleep(4 * time.Second)

				// re-create the "shouldTransfer" files, after our includeAfter time.
				fs := h.GetTestFiles().cloneShouldTransfers()
				h.CreateFiles(fs, true)
			},
		},
		testFiles{
			defaultSize: "1K",
			shouldIgnore: []interface{}{
				"filea",
			},
			shouldTransfer: []interface{}{
				"fileb",
			},
		})
}

func TestFilter_IncludePattern(t *testing.T) {

	RunScenarios(
		t,
		eOperation.Copy(),
		eTestFromTo.AllSourcesToOneDest(),
		eValidate.Auto(),
		params{
			recursive:      true,
			includePattern: "*.txt;2020*;*mid*;file8", //*pre*in*post*",
		},
		nil,
		testFiles{
			defaultSize: "1K",
			shouldIgnore: []interface{}{
				"A2020log",
				"A2020log.txte",
			},
			shouldTransfer: []interface{}{
				folder("subdir"),
				"2020_file1",
				"file2.txt",
				"file3_mid_txt",
				"subdir/2020_file5", //because recursive=true and patterns are matched in subdirectories as well.
				"subdir/file6.txt",
				"subdir/file7_A_mid_B",
				"file8", //Exact match
			},
		})
}

func TestFilter_RemoveFile(t *testing.T) {

	RunScenarios(
		t,
		eOperation.Remove(),
		eTestFromTo.AllRemove(),
		eValidate.Auto(),
		params{
			relativeSourcePath: "file2.txt",
		},
		nil,
		testFiles{
			defaultSize: "1K",
			shouldTransfer: []interface{}{
				"file1.txt",
			},
			shouldIgnore: []interface{}{
				"file2.txt",
			},
		})
}

func TestFilter_RemoveFolder(t *testing.T) {

	RunScenarios(
		t,
		eOperation.Remove(),
		eTestFromTo.AllRemove(),
		eValidate.Auto(),
		params{
			recursive:          true,
			relativeSourcePath: "folder2/",
		},
		nil,
		testFiles{
			defaultSize: "1K",
			shouldTransfer: []interface{}{
				"file1.txt",
				"folder1/file11.txt",
				"folder1/file12.txt",
			},
			shouldIgnore: []interface{}{
				"folder2/file21.txt",
				"folder2/file22.txt",
			},
		})
}

func TestFilter_RemoveContainer(t *testing.T) {

	RunScenarios(
		t,
		eOperation.Remove(),
		eTestFromTo.AllRemove(),
		eValidate.Auto(),
		params{
			recursive:          true,
			relativeSourcePath: "",
		},
		nil,
		testFiles{
			defaultSize: "1K",
			shouldTransfer: []interface{}{
				"file1.txt",
				"folder1/file11.txt",
				"folder1/file12.txt",
			},
		})
}

func TestFilter_ExcludePath(t *testing.T) {

	RunScenarios(
		t,
		eOperation.Copy(),
		eTestFromTo.AllSourcesToOneDest(),
		eValidate.Auto(),
		params{
			recursive:   true,
			excludePath: "subL1/subL2;excludeFile",
		},
		nil,
		testFiles{
			defaultSize: "1K",
			shouldIgnore: []interface{}{
				"excludeFile",
				folder("subL1/subL2"),
				"subL1/subL2/file1",
			},
			shouldTransfer: []interface{}{
				folder(""),
				folder("subL1"),
				folder("sub"),
				folder("subL1/sub"),
				folder("sub/subL1"),
				folder("subL1/sub/subL2"),
				folder("sub/subL1/subL2"),
				"sub/excludeFile",       // exclude path starts at root
				"subL1/sub/subL2/fileA", //exclude path should be contiguous
				"sub/subL1/subL2/fileB",
			},
		})
}

func TestFilter_ExcludePattern(t *testing.T) {

	RunScenarios(
		t,
		eOperation.Copy(),
		eTestFromTo.AllSourcesToOneDest(),
		eValidate.Auto(),
		params{
			recursive:      true,
			excludePattern: "*.log;2020*;*mid*;excludeFile",
		},
		nil,
		testFiles{
			defaultSize: "1K",
			shouldIgnore: []interface{}{
				"A2020.log",
				"2020log.txt",
				"A2020_mid_file",
				"excludeFile",
				"subdir/A2020.log", //We'll match patterns as sub-directories if recursive=true
				"subdir/2020log.txt",
				"subdir/A2020_mid_file",
			},
			shouldTransfer: []interface{}{
				folder("subdir"),
				"sample.txt",
				"subdir/sample.txt",
			},
		})
}