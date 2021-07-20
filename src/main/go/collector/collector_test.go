/*******************************************************************************
 * IBM Watson Imaging Common Application Framework 3.0                         *
 *                                                                             *
 * IBM Confidential                                                            *
 *                                                                             *
 * OCO Source Materials                                                        *
 *                                                                             *
 * (C) Copyright IBM Corp. 2019                                                *
 *                                                                             *
 * The source code for this program is not published or otherwise              *
 * divested of its trade secrets, irrespective of what has been                *
 * deposited with the U.S. Copyright Office.                                   *
 *******************************************************************************/
package main

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	shutdown()
	os.Exit(code)
}

func setup() {
}

func shutdown() {
}

func TestTrue(t *testing.T) {
	if !true {
		t.Error("false")
	}
}
