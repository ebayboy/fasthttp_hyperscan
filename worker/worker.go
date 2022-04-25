package worker

import log "github.com/sirupsen/logrus"

//接口: 继承此接口的模块需要实现接口函数
type Worker interface {
	//业务处理
	Process(interface{}) interface{}

	//控制同步
	BlockUntilReady()
	Interrupt()
	Terminate()
}

type WorkRequest struct {
	//业务接收通道
	JobChan chan<- interface{}

	//结果返回通道
	RetChan <-chan interface{}

	//中断函数
	InterruptFunc func()
}

type WorkerWrapper struct {
	//业务处理
	Worker Worker

	//业务接收通道
	ReqChan chan<- WorkRequest

	//控制通道
	InterruptChan chan struct{}
	CloseChan     chan struct{}
	ClosedChan    chan struct{}
}

func NewWorkerWrapper(reqChan chan<- WorkRequest, worker Worker) *WorkerWrapper {
	w := WorkerWrapper{
		Worker:        worker,
		ReqChan:       reqChan,
		InterruptChan: make(chan struct{}), /* 自初始化 */
		CloseChan:     make(chan struct{}), /* 自初始化 */
		ClosedChan:    make(chan struct{}), /* 自初始化 */
	}

	log.Println("NewWorkerWrapper  before w.Run , ClosedChan:", w.ClosedChan)
	go w.run()

	return &w
}

func (w *WorkerWrapper) Interrupt() {
	close(w.InterruptChan)
	w.Worker.Interrupt()
}

func (w *WorkerWrapper) run() {

	//TODO: jobChan, retChan是内部创建的， 如何写入
	jobChan, retChan := make(chan interface{}), make(chan interface{})

	//defere : Run退出后，关闭通道
	defer func() {
		w.Worker.Terminate() //最终调用实现此接口的函数closureWorker->Terminate()
		log.Info("defer close(retChan):", retChan)
		close(retChan)

		//TODO: panic: close of closed channel
		log.Info("Stop -> CloseChan -> defer close(w.ClosedChan):", w.ClosedChan)
		close(w.ClosedChan)
	}()

	for {
		//BlockUntilReady执行完成后开始处理任务
		w.Worker.BlockUntilReady()
		log.Println("w.Worker.BlockUntilRead done! Next will block with select...")

		select {
		case w.ReqChan <- WorkRequest{
			JobChan:       jobChan,
			RetChan:       retChan,
			InterruptFunc: w.Interrupt,
		}:
			log.Println("Run Read success: w.ReqChan <- WorkRequest: JobChan")
			select {
			//从jobChan读取到payload
			case payload := <-jobChan:
				log.Println("Run jobChan read payload:", payload)
				//调用Process
				result := w.Worker.Process(payload)
				select {
				//将result写入到retChan
				case retChan <- result:
					log.Println("Run retChan <- result")
					//case + 初始化
				case <-w.InterruptChan:
					w.InterruptChan = make(chan struct{})
					log.Println("Run -w.InterruptChan")
				}
			case <-w.InterruptChan:
				w.InterruptChan = make(chan struct{})
			}

			//从CloseChan读取到消息, 退出Run
		case <-w.CloseChan:
			log.Info("recv <- w.CloseChan: ", w.ClosedChan)
			return
		}
	}
}

func (w *WorkerWrapper) Stop() {
	log.Info("Stop w.CloseChan:", w.CloseChan)
	close(w.CloseChan)
}

func (w *WorkerWrapper) Join() {
	log.Info("Join w.ClosedChan:", w.ClosedChan)
	<-w.ClosedChan
}
