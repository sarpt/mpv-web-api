package mpv_test

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/sarpt/mpv-web-api/internal/mocks"
	"github.com/sarpt/mpv-web-api/pkg/mpv"
)

func TestNext_MultiplePayloadsInOneRead(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// given
	conn := mocks.NewMockConn(ctrl)
	conn.
		EXPECT().
		Read(gomock.Any()).
		DoAndReturn(func(buf []byte) (int, error) {
			result := []byte("{\"Name\":\"test\",\"Data\":5}\n{\"Name\":\"event\"}\n")

			return copy(buf, result), nil
		}).
		Times(1)

	uut := mpv.NewResponsesIterator(conn)
	if uut == nil {
		t.Fatalf("Response iterator is nil")
	}

	// when
	response1, err := uut.Next()
	if err != nil {
		t.Fatalf("Unexpected error reported for response1: %s", err)
	}

	response2, err := uut.Next()
	if err != nil {
		t.Fatalf("Unexpected error reported for response2: %s", err)
	}

	// then
	expectedName1 := "test"
	name1 := response1.Name
	if name1 != expectedName1 {
		t.Errorf("Expected name %s to equal %s", name1, expectedName1)
	}

	expectedData1 := float64(5)
	data1, ok := response1.Data.(float64)
	if !ok {
		t.Fatalf("Cannot cast data in response 1 to float64")
	}

	if data1 != expectedData1 {
		t.Errorf("Expected data %f to equal %f", data1, expectedData1)
	}

	expectedName2 := "event"
	name2 := response2.Name
	if name2 != expectedName2 {
		t.Errorf("Expected name %s to equal %s", name1, expectedName1)
	}
}

func TestNext_OnePayloadInMultipleReads(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// given
	conn := mocks.NewMockConn(ctrl)
	firstReadCall := conn.
		EXPECT().
		Read(gomock.Any()).
		DoAndReturn(func(buf []byte) (int, error) {
			result := []byte("{\"Name\":\n")

			return copy(buf, result), nil
		}).
		Times(1)

	secondReadCall := conn.
		EXPECT().
		Read(gomock.Any()).
		DoAndReturn(func(buf []byte) (int, error) {
			result := []byte("\"test\",")

			return copy(buf, result), nil
		}).
		Times(1).
		After(firstReadCall)

	conn.
		EXPECT().
		Read(gomock.Any()).
		DoAndReturn(func(buf []byte) (int, error) {
			result := []byte("\"Data\":5}\n{\"Name\":\"event\"}\n")

			return copy(buf, result), nil
		}).
		Times(1).
		After(secondReadCall)

	uut := mpv.NewResponsesIterator(conn)
	if uut == nil {
		t.Fatalf("Response iterator is nil")
	}

	// when
	response1, err := uut.Next()
	if err != nil {
		t.Fatalf("Unexpected error reported for response1: %s", err)
	}

	response2, err := uut.Next()
	if err != nil {
		t.Fatalf("Unexpected error reported for response2: %s", err)
	}

	// then
	expectedName1 := "test"
	name1 := response1.Name
	if name1 != expectedName1 {
		t.Errorf("Expected name %s to equal %s", name1, expectedName1)
	}

	expectedData1 := float64(5)
	data1, ok := response1.Data.(float64)
	if !ok {
		t.Fatalf("Cannot cast data in response 1 to float64")
	}

	if data1 != expectedData1 {
		t.Errorf("Expected data %f to equal %f", data1, expectedData1)
	}

	expectedName2 := "event"
	name2 := response2.Name
	if name2 != expectedName2 {
		t.Errorf("Expected name %s to equal %s", name1, expectedName1)
	}
}

func TestNext_ConsecutiveNewlines(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// given
	conn := mocks.NewMockConn(ctrl)
	conn.
		EXPECT().
		Read(gomock.Any()).
		DoAndReturn(func(buf []byte) (int, error) {
			result := []byte("\n{\"Name\":\"test\",\"Data\":5}\n\n\n\n{\"Name\":\"event\"}\n\n")

			return copy(buf, result), nil
		}).
		Times(1)

	uut := mpv.NewResponsesIterator(conn)
	if uut == nil {
		t.Fatalf("Response iterator is nil")
	}

	// when
	response1, err := uut.Next()
	if err != nil {
		t.Fatalf("Unexpected error reported for response1: %s", err)
	}

	response2, err := uut.Next()
	if err != nil {
		t.Fatalf("Unexpected error reported for response2: %s", err)
	}

	// then
	expectedName1 := "test"
	name1 := response1.Name
	if name1 != expectedName1 {
		t.Errorf("Expected name %s to equal %s", name1, expectedName1)
	}

	expectedData1 := float64(5)
	data1, ok := response1.Data.(float64)
	if !ok {
		t.Fatalf("Cannot cast data in response 1 to float64")
	}

	if data1 != expectedData1 {
		t.Errorf("Expected data %f to equal %f", data1, expectedData1)
	}

	expectedName2 := "event"
	name2 := response2.Name
	if name2 != expectedName2 {
		t.Errorf("Expected name %s to equal %s", name1, expectedName1)
	}
}
