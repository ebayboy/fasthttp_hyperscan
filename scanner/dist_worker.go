package scanner

import (
	"context"
	"math/rand"

	"github.com/Jeffail/tunny"
	log "github.com/sirupsen/logrus"
)

type DistWorkerContext struct {
	Data       map[string][]byte //map[data_key]data
	DistWorker *DistWorker
}

type DistWorker struct {
	NumScanWorker int
	ScanWorkers   []*ScanWorker  //scanWorker -> scanner
	Pool          *tunny.Pool 
}

//tunny.pool 多协程同时调用此函数, 要保证线程安全
//main -> dist_worker -> scanner_worker -> scanner -> matcher
//w.Pool.Process -> distWorkerCallback
func distWorkerCallback(distWorkerContext interface{}) (err interface{}) {

	log.Debug("distWorkerCallback...")

	ctx := distWorkerContext.(*DistWorkerContext)
	scanCtx := ScanWorkerContext{Data: ctx.Data}

	//将任务随机分发到scanner_worker
	idx := rand.Intn(ctx.DistWorker.NumScanWorker)
	log.Debugf("Select ScanWorkers[%d]", idx)
	if err = ctx.DistWorker.ScanWorkers[idx].Scan(&scanCtx); err != nil {
		log.Error("Error! err:", err.(error).Error())
		return nil
	}

	return scanCtx.Res
}

func NewDistWorker(numScanWorker int, confData []byte, mctx *context.Context, cf *Conf) (*DistWorker, error) {

	dist := &DistWorker{NumScanWorker: numScanWorker}
	
	//初始化goroute协程池
	dist.Pool = tunny.NewFunc(numScanWorker, distWorkerCallback)

	for i := 0; i < numScanWorker; i++ {
		scan_worker, err := NewScanWorker(confData, mctx, cf)
		if err != nil {
			return nil, err
		}
		dist.ScanWorkers = append(dist.ScanWorkers, scan_worker)
		log.Infof("NewScanWorker[%d]", i)
	}

	return dist, nil
}

//TODO: 好像没走到这个函数
func (w *DistWorker) Process(distWorkerCtx interface{}) (res interface{}) {

	log.Debug("DistWorker.Process...")

	//从协程池中自动选择协程执行回调函数distWorkerCallback
	res = w.Pool.Process(distWorkerCtx)  // w.Pool.Process -> distWorkerCallback

	return res
}

func (w *DistWorker) Stop() {
	log.Info("DistWorker Stop...")
	if w.Pool != nil {
		w.Pool.Close()
	}

	for k, _ := range w.ScanWorkers {
		w.ScanWorkers[k].Stop()
	}
	log.Info("DistWorker Stoped!")
}
