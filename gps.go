/*
 * talkkonnect headless mumble client/gateway with lcd screen and channel control
 * Copyright (C) 2018-2019, Suvir Kumar <suvir@talkkonnect.com>
 *
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/.
 *
 * Software distributed under the License is distributed on an "AS IS" basis,
 * WITHOUT WARRANTY OF ANY KIND, either express or implied. See the License
 * for the specific language governing rights and limitations under the
 * License.
 *
 * talkkonnect is the based on talkiepi and barnard by Daniel Chote and Tim Cooper
 *
 * The Initial Developer of the Original Code is
 * Suvir Kumar <suvir@talkkonnect.com>
 * Portions created by the Initial Developer are Copyright (C) Suvir Kumar. All Rights Reserved.
 *
 * Contributor(s):
 *
 * Suvir Kumar <suvir@talkkonnect.com>
 *
 * My Blog is at www.talkkonnect.com
 * The source code is hosted at github.com/talkkonnect
 *
 * gps.go -> talkkonnect function to interface to USB GPS Neo6M
 */

package talkkonnect

import (
	"bufio"
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/jacobsa/go-serial/serial"
	"github.com/pacomont/go-nmea"
)

var (
	sendToTracCar  bool   = true
	tracCarURL     string = "http://demo.traccar.org"
	tracCarPort    string = "5060"
	tracCarID      string = "12345"
	tracCarFullURL string = ""
)

var goodGPSRead bool = false

func getGpsPosition(verbose bool) (bool, error) {
	if GpsEnabled {

		if Port == "" {
			return false, errors.New("You Must Specify Port")
		}

		if Even && Odd {
			return false, errors.New("can't specify both even and odd parity")
		}

		parity := serial.PARITY_NONE

		if Even {
			parity = serial.PARITY_EVEN
		} else if Odd {
			parity = serial.PARITY_ODD
		}

		options := serial.OpenOptions{
			PortName:               Port,
			BaudRate:               Baud,
			DataBits:               DataBits,
			StopBits:               StopBits,
			MinimumReadSize:        MinRead,
			InterCharacterTimeout:  CharTimeOut,
			ParityMode:             parity,
			Rs485Enable:            Rs485,
			Rs485RtsHighDuringSend: Rs485HighDuringSend,
			Rs485RtsHighAfterSend:  Rs485HighAfterSend,
		}

		f, err := serial.Open(options)
		defer f.Close()

		if err != nil {
			GpsEnabled = false
			return false, errors.New("Cannot Open Serial Port")
		}

		if TxData != "" {
			txData, err := hex.DecodeString(TxData)

			if err != nil {
				GpsEnabled = false
				return false, errors.New("error: Cannot Decode Hex Data")
			}

			log.Println("debug: Sending: ", hex.EncodeToString(txData))

			count, err := f.Write(txData)

			if err != nil {
				return false, errors.New("error: Error writing to serial port")
			}
			log.Println("debug: Wrote %v bytes\n", count)
		}

		if Rx {
			serialPort, err := serial.Open(options)
			if err != nil {
				log.Println("error: Unable to Open Serial Port Error ", err)
			}

			defer serialPort.Close()

			reader := bufio.NewReader(serialPort)
			scanner := bufio.NewScanner(reader)

			goodGPSRead = false
			for scanner.Scan() {
				s, err := nmea.Parse(scanner.Text())

				if err == nil {
					if s.DataType() == nmea.TypeRMC {
						m := s.(nmea.RMC)
						if m.Latitude != 0 && m.Longitude != 0 {
							goodGPSRead = true
							tracCarFullURL = fmt.Sprintf(tracCarURL + ":" + tracCarPort + "?" + "id=" + tracCarID + "&" + "lat={" + nmea.FormatGPS(m.Latitude) + "}" + "&" + "lon={" + nmea.FormatGPS(m.Longitude) + "}" + "&" + "timestamp={" + GPSTime + "}" + "&" + "hdop={" + "}" + "&a")
							GPSTime = fmt.Sprintf("%v", m.Time)
							GPSDate = fmt.Sprintf("%v", m.Date)
							GPSLatitude = m.Latitude
							GPSLongitude = m.Longitude
							if verbose {
								log.Println("info: Time: ", m.Time)
								log.Println("info: Validity: ", m.Validity)
								log.Println("info: Latitude GPS: ", nmea.FormatGPS(m.Latitude))
								log.Println("info: Latitude DMS: ", nmea.FormatDMS(m.Latitude))
								log.Println("info: Longitude GPS: ", nmea.FormatGPS(m.Longitude))
								log.Println("info: Longitude DMS: ", nmea.FormatDMS(m.Longitude))
								log.Println("info: Speed: ", m.Speed)
								log.Println("info: Course: ", m.Course)
								log.Println("info: Date: ", m.Date)
								log.Println("info: Variation: ", m.Variation)
								log.Println(tracCarFullURL)

							}
							if sendToTracCar {
								httpSendTracCar()
							}
							break
						} else {
							log.Println("error: Got Latitude 0 and Longtitude 0 from GPS")
						}
					} else {
						log.Println("error: GPS Sentence Format Was not nmea.RMC")
					}
				} else {
					log.Println("error: Scanner Function Error ", err)
				}
			}
		} else {
			return false, errors.New("Rx Not Set")
		}

		return goodGPSRead, nil
	}
	return false, errors.New("GPS Not Enabled")
}

func httpSendTracCar() {
	response, err := http.Get(tracCarFullURL)
	if err != nil {
		log.Println("error: Cannot Read traccar Webpage! Error ", err)
	} else {
		defer response.Body.Close()
		contents, err := ioutil.ReadAll(response.Body)
		if err != nil {
			log.Println("error: Error Sending Data to traccar Webpage!")
		} else {
			log.Println("debug: traccar web response ", string(contents))
		}
	}
	return
}
