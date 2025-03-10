// Copyright 2021 TiKV Project Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package core

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	. "github.com/pingcap/check"
)

type MockPackHotRegionInfo struct {
	isLeader         bool
	historyHotReads  []HistoryHotRegion
	historyHotWrites []HistoryHotRegion
}

// PackHistoryHotWriteRegions get read hot region info in HistoryHotRegion from.
func (m *MockPackHotRegionInfo) PackHistoryHotReadRegions() ([]HistoryHotRegion, error) {
	return m.historyHotReads, nil
}

// PackHistoryHotWriteRegions get write hot region info in HistoryHotRegion form.
func (m *MockPackHotRegionInfo) PackHistoryHotWriteRegions() ([]HistoryHotRegion, error) {
	return m.historyHotWrites, nil
}

// IsLeader return isLeader.
func (m *MockPackHotRegionInfo) IsLeader() bool {
	return m.isLeader
}

// GenHistoryHotRegions generate history hot region for test.
func (m *MockPackHotRegionInfo) GenHistoryHotRegions(num int, updateTime time.Time) {
	for i := 0; i < num; i++ {
		historyHotRegion := HistoryHotRegion{
			UpdateTime:    updateTime.UnixNano() / int64(time.Millisecond),
			RegionID:      uint64(i),
			StoreID:       uint64(i),
			PeerID:        rand.Uint64(),
			IsLeader:      i%2 == 0,
			IsLearner:     i%2 == 0,
			HotRegionType: HotRegionTypes[i%2],
			HotDegree:     int64(rand.Int() % 100),
			FlowBytes:     rand.Float64() * 100,
			KeyRate:       rand.Float64() * 100,
			QueryRate:     rand.Float64() * 100,
			StartKey:      fmt.Sprintf("%20d", i),
			EndKey:        fmt.Sprintf("%20d", i),
		}
		if i%2 == 1 {
			m.historyHotWrites = append(m.historyHotWrites, historyHotRegion)
		} else {
			m.historyHotReads = append(m.historyHotReads, historyHotRegion)
		}
	}
}

// ClearHotRegion delete all region cached.
func (m *MockPackHotRegionInfo) ClearHotRegion() {
	m.historyHotReads = make([]HistoryHotRegion, 0)
	m.historyHotWrites = make([]HistoryHotRegion, 0)
}

var _ = Suite(&testHotRegionStorage{})

type testHotRegionStorage struct {
	ctx    context.Context
	cancel context.CancelFunc
}

func (t *testHotRegionStorage) SetUpSuite(c *C) {
	t.ctx, t.cancel = context.WithCancel(context.Background())
}

func (t *testHotRegionStorage) TestHotRegionWrite(c *C) {
	packHotRegionInfo := &MockPackHotRegionInfo{}
	store, clean, err := newTestHotRegionStorage(10*time.Minute, 1, packHotRegionInfo)
	c.Assert(err, IsNil)
	defer clean()
	now := time.Now()
	hotRegionStorages := []HistoryHotRegion{
		{
			UpdateTime:    now.UnixNano() / int64(time.Millisecond),
			RegionID:      1,
			StoreID:       1,
			HotRegionType: ReadType.String(),
			StartKey:      string([]byte{0x74, 0x80, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xff, 0x15, 0x5f, 0x69, 0x80, 0x0, 0x0, 0x0, 0x0, 0xff, 0x0, 0x0, 0x1, 0x0, 0x0, 0x0, 0x0, 0x0, 0xfa}),
			EndKey:        string([]byte{0x74, 0x80, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xff, 0x15, 0x5f, 0x69, 0x80, 0x0, 0x0, 0x0, 0x0, 0xff, 0x0, 0x0, 0x1, 0x0, 0x0, 0x0, 0x0, 0x0, 0xfa}),
		},
		{
			UpdateTime:    now.Add(10*time.Second).UnixNano() / int64(time.Millisecond),
			RegionID:      2,
			StoreID:       1,
			HotRegionType: ReadType.String(),
			StartKey:      string([]byte{0x74, 0x80, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xff, 0x15, 0x5f, 0x69, 0x80, 0x0, 0x0, 0x0, 0x0, 0xff, 0x0, 0x0, 0x2, 0x0, 0x0, 0x0, 0x0, 0x0, 0xfa}),
			EndKey:        string([]byte{0x74, 0x80, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xff, 0x15, 0x5f, 0x69, 0x80, 0x0, 0x0, 0x0, 0x0, 0xff, 0x0, 0x0, 0x2, 0x0, 0x0, 0x0, 0x0, 0x0, 0xfa}),
		},
		{
			UpdateTime:    now.Add(20*time.Second).UnixNano() / int64(time.Millisecond),
			RegionID:      3,
			StoreID:       1,
			HotRegionType: ReadType.String(),
			StartKey:      string([]byte{0x74, 0x80, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xff, 0x83, 0x5f, 0x69, 0x80, 0x0, 0x0, 0x0, 0x0, 0xff, 0x0, 0x0, 0x1, 0x0, 0x0, 0x0, 0x0, 0x0, 0xfa}),
			EndKey:        string([]byte{0x74, 0x80, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xff, 0x83, 0x5f, 0x69, 0x80, 0x0, 0x0, 0x0, 0x0, 0xff, 0x0, 0x0, 0x1, 0x0, 0x0, 0x0, 0x0, 0x0, 0xfa}),
		},
	}
	var copyHotRegionStorages []HistoryHotRegion
	data, _ := json.Marshal(hotRegionStorages)
	json.Unmarshal(data, &copyHotRegionStorages)
	for i, region := range hotRegionStorages {
		copyHotRegionStorages[i].StartKey = region.StartKey
		copyHotRegionStorages[i].EndKey = region.EndKey
	}
	packHotRegionInfo.historyHotReads = hotRegionStorages
	packHotRegionInfo.historyHotWrites = []HistoryHotRegion{
		{
			UpdateTime:    now.Add(30*time.Second).UnixNano() / int64(time.Millisecond),
			RegionID:      4,
			StoreID:       1,
			HotRegionType: WriteType.String(),
		},
	}
	store.pullHotRegionInfo()
	store.flush()
	iter := store.NewIterator([]string{ReadType.String()},
		now.UnixNano()/int64(time.Millisecond),
		now.Add(40*time.Second).UnixNano()/int64(time.Millisecond))
	index := 0
	for next, err := iter.Next(); next != nil && err == nil; next, err = iter.Next() {
		copyHotRegionStorages[index].StartKey = HexRegionKeyStr([]byte(copyHotRegionStorages[index].StartKey))
		copyHotRegionStorages[index].EndKey = HexRegionKeyStr([]byte(copyHotRegionStorages[index].EndKey))
		c.Assert(reflect.DeepEqual(&copyHotRegionStorages[index], next), IsTrue)
		index++
	}
	c.Assert(err, IsNil)
	c.Assert(index, Equals, 3)
}

func (t *testHotRegionStorage) TestHotRegionDelete(c *C) {
	defaultReaminDay := 7
	defaultDelteData := 30
	deleteDate := time.Now().AddDate(0, 0, 0)
	packHotRegionInfo := &MockPackHotRegionInfo{}
	store, clean, err := newTestHotRegionStorage(10*time.Minute, int64(defaultReaminDay), packHotRegionInfo)
	c.Assert(err, IsNil)
	defer clean()
	historyHotRegions := make([]HistoryHotRegion, 0)
	for i := 0; i < defaultDelteData; i++ {
		historyHotRegion := HistoryHotRegion{
			UpdateTime:    deleteDate.UnixNano() / int64(time.Millisecond),
			RegionID:      1,
			HotRegionType: ReadType.String(),
		}
		historyHotRegions = append(historyHotRegions, historyHotRegion)
		deleteDate = deleteDate.AddDate(0, 0, -1)
	}
	packHotRegionInfo.historyHotReads = historyHotRegions
	store.pullHotRegionInfo()
	store.flush()
	store.delete()
	iter := store.NewIterator(HotRegionTypes,
		deleteDate.UnixNano()/int64(time.Millisecond),
		time.Now().UnixNano()/int64(time.Millisecond))
	num := 0
	for next, err := iter.Next(); next != nil && err == nil; next, err = iter.Next() {
		num++
		c.Assert(reflect.DeepEqual(next, &historyHotRegions[defaultReaminDay-num]), IsTrue)
	}
}

func BenchmarkInsert(b *testing.B) {
	packHotRegionInfo := &MockPackHotRegionInfo{}
	regionStorage, clear, err := newTestHotRegionStorage(10*time.Hour, 7, packHotRegionInfo)
	defer clear()
	if err != nil {
		b.Fatal(err)
	}
	packHotRegionInfo.GenHistoryHotRegions(1000, time.Now())
	b.ResetTimer()
	regionStorage.pullHotRegionInfo()
	regionStorage.flush()
	b.StopTimer()
}

func BenchmarkInsertAfterMonth(b *testing.B) {
	defaultInsertDay := 30
	packHotRegionInfo := &MockPackHotRegionInfo{}
	regionStorage, clear, err := newTestHotRegionStorage(10*time.Hour, int64(defaultInsertDay), packHotRegionInfo)
	defer clear()
	if err != nil {
		b.Fatal(err)
	}
	nextTime := newTestHotRegions(regionStorage, packHotRegionInfo, 144*defaultInsertDay, 1000, time.Now())
	packHotRegionInfo.GenHistoryHotRegions(1000, nextTime)
	b.ResetTimer()
	regionStorage.pullHotRegionInfo()
	regionStorage.flush()
	b.StopTimer()
}

func BenchmarkDelete(b *testing.B) {
	defaultInsertDay := 7
	defaultReaminDay := 7
	packHotRegionInfo := &MockPackHotRegionInfo{}
	regionStorage, clear, err := newTestHotRegionStorage(10*time.Hour, int64(defaultReaminDay), packHotRegionInfo)
	defer clear()
	if err != nil {
		b.Fatal(err)
	}
	deleteTime := time.Now().AddDate(0, 0, -14)
	newTestHotRegions(regionStorage, packHotRegionInfo, 144*defaultInsertDay, 1000, deleteTime)
	b.ResetTimer()
	regionStorage.delete()
	b.StopTimer()
}

func BenchmarkRead(b *testing.B) {
	packHotRegionInfo := &MockPackHotRegionInfo{}
	regionStorage, clear, err := newTestHotRegionStorage(10*time.Hour, 7, packHotRegionInfo)
	if err != nil {
		b.Fatal(err)
	}
	defer clear()
	endTime := time.Now()
	startTime := endTime
	endTime = newTestHotRegions(regionStorage, packHotRegionInfo, 144*7, 1000, endTime)
	b.ResetTimer()
	iter := regionStorage.NewIterator(HotRegionTypes, startTime.UnixNano()/int64(time.Millisecond),
		endTime.AddDate(0, 1, 0).UnixNano()/int64(time.Millisecond))
	next, err := iter.Next()
	for next != nil && err == nil {
		next, err = iter.Next()
	}
	if err != nil {
		b.Fatal(err)
	}
	b.StopTimer()
}

func newTestHotRegions(storage *HotRegionStorage, mock *MockPackHotRegionInfo, cycleTimes, num int, updateTime time.Time) time.Time {
	for i := 0; i < cycleTimes; i++ {
		mock.GenHistoryHotRegions(num, updateTime)
		storage.pullHotRegionInfo()
		storage.flush()
		updateTime = updateTime.Add(10 * time.Minute)
		mock.ClearHotRegion()
	}
	return updateTime
}

func newTestHotRegionStorage(pullInterval time.Duration,
	remianedDays int64,
	packHotRegionInfo HotRegionStorageHandler) (
	hotRegionStorage *HotRegionStorage,
	clear func(), err error) {
	writePath := "./tmp"
	ctx := context.Background()
	if err != nil {
		return nil, nil, err
	}
	// delete data in between today and tomrrow
	hotRegionStorage, err = NewHotRegionsStorage(ctx,
		writePath, nil, packHotRegionInfo, remianedDays, pullInterval)
	if err != nil {
		return nil, nil, err
	}
	clear = func() {
		hotRegionStorage.Close()
		PrintDirSize(writePath)
		os.RemoveAll(writePath)
	}
	return
}

// Print dir size
func PrintDirSize(path string) {
	size, err := DirSizeB(path)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("file size %d\n", size)
}

// DirSizeB get file size by path(B)
func DirSizeB(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			size += info.Size()
		}
		return err
	})
	return size, err
}
