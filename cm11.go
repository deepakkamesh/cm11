/* Package cm11 implements the driver interface to communicate with
x10 CM11 controller
*/
package cm11

import (
	"fmt"
	"io"
	"log"
	"time"

	"github.com/tarm/serial"
)

type ObjState struct {
	HouseCode    string
	DeviceCode   string
	FunctionCode string
}

type Device struct {
	serialPort    string
	serial        *serial.Port
	out           chan ObjState
	in            chan ObjState
	houseCode     map[string]byte
	deviceCode    map[string]byte
	functionCode  map[string]byte
	houseCodeR    map[byte]string
	deviceCodeR   map[byte]string
	functionCodeR map[byte]string
}

func (m *Device) Init() error {
	log.Printf("initialize device %s", m.serialPort)
	c := &serial.Config{Name: m.serialPort, Baud: 4800, ReadTimeout: time.Millisecond * 500}
	s, err := serial.OpenPort(c)
	if err != nil {
		return fmt.Errorf("error opening device %s", err)
	}

	// Flush any stale data by requesting a transmission.
	s.Write([]byte{0xC3})
	m.serial = s
	go m.run()
	return nil
}

func New(serialPort string, c chan ObjState) *Device {

	houseCode := map[string]byte{
		"A": 0x06,
		"B": 0x0E,
		"C": 0x02,
		"D": 0x0A,
		"E": 0x01,
		"F": 0x09,
		"G": 0x05,
		"H": 0x0D,
		"I": 0x07,
		"J": 0x0F,
		"K": 0x03,
		"L": 0x0B,
		"M": 0x00,
		"N": 0x08,
		"O": 0x04,
		"P": 0x0C,
	}

	deviceCode := map[string]byte{
		"1":  0x06,
		"2":  0x0E,
		"3":  0x02,
		"4":  0x0A,
		"5":  0x01,
		"6":  0x09,
		"7":  0x05,
		"8":  0x0D,
		"9":  0x07,
		"10": 0x0F,
		"11": 0x03,
		"12": 0x0B,
		"13": 0x00,
		"14": 0x08,
		"15": 0x04,
		"16": 0x0C,
	}

	functionCode := map[string]byte{
		"All Units Off":          0x00,
		"All Lights On":          0x01,
		"On":                     0x02,
		"Off":                    0x03,
		"Dim":                    0x04,
		"Bright":                 0x05,
		"All Lights Off":         0x06,
		"Extended Code":          0x07,
		"Hail Request":           0x08,
		"Hail Acknoledge":        0x09,
		"Pre-set Dim_1":          0x0A,
		"Pre-set Dim_2":          0x0B,
		"Extended Data Transfer": 0x0C,
		"Status On":              0x0D,
		"Status Off":             0x0E,
		"Status Request":         0x0F,
	}

	// Setup reverse lookup maps
	houseCodeR := make(map[byte]string, len(houseCode))
	deviceCodeR := make(map[byte]string, len(deviceCode))
	functionCodeR := make(map[byte]string, len(functionCode))

	for k, v := range houseCode {
		houseCodeR[v] = k
	}
	for k, v := range deviceCode {
		deviceCodeR[v] = k
	}
	for k, v := range functionCode {
		functionCodeR[v] = k
	}

	return &Device{
		serialPort:    serialPort,
		out:           c,
		in:            make(chan ObjState, 10),
		houseCode:     houseCode,
		deviceCode:    deviceCode,
		functionCode:  functionCode,
		houseCodeR:    houseCodeR,
		deviceCodeR:   deviceCodeR,
		functionCodeR: functionCodeR,
	}
}

func (m *Device) SendCommand(house string, device string, function string) {
	m.in <- ObjState{
		HouseCode:    house,
		DeviceCode:   device,
		FunctionCode: function,
	}
}

// run is the main processing loop.
func (m *Device) run() {
	tick := time.NewTicker(time.Millisecond * 200)

	for {
		buf := make([]byte, 20)
		_, err := m.serial.Read(buf)

		// Line is free to write commands.
		if err == io.EOF {
			select {
			case o := <-m.in:
				// Write house+device code.
				b := m.houseCode[o.HouseCode]<<4 + m.deviceCode[o.DeviceCode]
				if err := m.writeCmd([]byte{0x04, b}); err != nil {
					log.Printf("command %s%s%s %s", o.HouseCode, o.DeviceCode, o.FunctionCode, err)
					continue
				}
				// Write house+function code.
				b = m.houseCode[o.HouseCode]<<4 + m.functionCode[o.FunctionCode]
				if err := m.writeCmd([]byte{0x06, b}); err != nil {
					log.Printf("command %s%s%s %s", o.HouseCode, o.DeviceCode, o.FunctionCode, err)
					continue
				}
				log.Printf("sent cm11 command %s%s-%s", o.HouseCode, o.DeviceCode, o.FunctionCode)
				continue
			case <-tick.C:
				continue
			}
		}

		// Data available to be read.
		if buf[0] == 0x5A {
			data, err := m.readCmd()
			if err != nil {
				log.Printf("skipping bad data transmission due to %s", err)
				continue
			}
			// Translate the transmission and send on channel.
			for i := 0; i < len(data[1:]); i++ {
				mask := (data[0] >> uint(i)) & 0x01 // Address or function mask.
				if mask == 0 && i < len(data[1:])-1 {
					m.out <- ObjState{
						HouseCode:    m.houseCodeR[data[i+1]>>4],
						DeviceCode:   m.deviceCodeR[(data[i+1]<<4)>>4],
						FunctionCode: m.functionCodeR[(data[i+2]<<4)>>4],
					}
					log.Printf("recieved cm11 command %s%s-%s",
						m.houseCodeR[data[i+1]>>4], m.deviceCodeR[(data[i+1]<<4)>>4], m.functionCodeR[(data[i+2]<<4)>>4])
					i += 1 // Skip the next byte.
				}
			}
		}
	}
}

// writeCmd writes the byte array command to cm11.
func (m *Device) writeCmd(b []byte) error {
	buf := make([]byte, 20)
	var (
		err error
		sum byte
	)

	// Write command and retry if checksum fails
	for {
		if _, err = m.serial.Write(b); err != nil {
			return fmt.Errorf("%q send failure %s", b, err)
		}
		// Read checksum.
		time.Sleep(500 * time.Millisecond)
		if _, err = m.serial.Read(buf); err != nil {
			return fmt.Errorf("checksum read failure %s", err)
		}
		// Calculate and verify checksum.
		for _, v := range b {
			sum += v
		}
		if buf[0] == byte(sum&0xFF) {
			// Ok to send transmission online.
			if _, err = m.serial.Write([]byte{0x00}); err != nil {
				return fmt.Errorf("0x00 send failure %s", err)
			}
			// Verify if interface is ready. Should return 0x55.
			time.Sleep(500 * time.Millisecond)
			if _, err = m.serial.Read(buf); err != nil {
				return fmt.Errorf("ready read failure %s", err)
			}
			break
		}
		return fmt.Errorf("checksum failure. Retry...")
	}
	return nil
}

// readCmd reads the available command from cm11.
func (m *Device) readCmd() ([]byte, error) {
	buf := make([]byte, 10)
	cmd := []byte{}

	// Ack the waiting data. 0xC3 starts the transmission.
	if _, err := m.serial.Write([]byte{0xC3}); err != nil {
		return nil, fmt.Errorf("write failure for 0xC3 %s", err)
	}

	// Read the data packet size.
	if _, err := m.serial.Read(buf); err != nil {
		return nil, fmt.Errorf("read failure %s", err)
	}
	dz := int(buf[0])

	// Read the  transmission.
	for i := 0; i < dz; i++ {
		n, err := m.serial.Read(buf)
		if err != nil {
			return nil, fmt.Errorf("read failure %s", err)
		}
		cmd = append(cmd, buf[:n]...)
	}
	return cmd, nil
}
