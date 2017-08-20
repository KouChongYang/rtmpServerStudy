package rtmp
import	"rtmpServerStudy/avQueue"

func (self *avQueue.AvQueue) Cl1ose() (err error) {
	self.lock.Lock()

	self.closed = true
	self.cond.Signal()

	self.lock.Unlock()
	return
}
