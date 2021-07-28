package main

import (
	"context"
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/color"
	"log"
	"time"

	pb "github.com/mailgun/kafka-pixy/gen/golang"
	"gocv.io/x/gocv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var (
	deviceID       = flag.Int("device", 1, "camera device ID")
	captureRate    = flag.Duration("rate", time.Second, "image capture rate (duration)")
	classifierPath = flag.String("classifier", "haarcascade_frontalface_default.xml", "path to the face recognition classifier")

	meroxaTLS      = flag.Bool("meroxa.tls", false, "enables/disables TLS connection to Meroxa gRPC endpoint")
	meroxaEndpoint = flag.String("meroxa.endpoint", "endpoint-grpc.meroxa.io:80", "URL to Meroxa gRPC endpoint")
	meroxaUsername = flag.String("meroxa.username", "", "Meroxa gRPC username")
	meroxaPassword = flag.String("meroxa.password", "", "Meroxa gRPC password")
	meroxaStream   = flag.String("meroxa.stream", "", "Meroxa stream")
)

func main() {
	flag.Parse()

	log.Printf("opening connection to Meroxa gRPC endpoint %s\n", *meroxaEndpoint)
	exporter, err := NewImageExporter(*meroxaEndpoint, *meroxaTLS, *meroxaUsername, *meroxaPassword, *meroxaStream)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("opening video capture on device %d\n", *deviceID)
	faceRecognizer, err := NewFaceRecognizer(*deviceID, *classifierPath)
	if err != nil {
		log.Fatal(err)
	}
	defer faceRecognizer.Close()

	err = run(faceRecognizer, exporter)
	if err != nil {
		log.Fatal(err)
	}
}

func run(faceRecognizer *FaceRecognizer, exporter *ImageExporter) error {
	// open display window
	window := gocv.NewWindow("Face Detect")
	defer window.Close()

	rgb := color.RGBA{0, 0, 255, 0}

	for {
		img, rects, err := faceRecognizer.Detect(*captureRate)
		if err != nil {
			return fmt.Errorf("error while detecting face: %w", err)
		}

		log.Printf("detected %d faces\n", len(rects))
		// draw a rectangle around each face on the original image
		for _, r := range rects {
			gocv.Rectangle(img, r, rgb, 3)
		}

		err = exporter.Send(img)
		if err != nil {
			return fmt.Errorf("could not export image: %w", err)
		}

		// show the image in the window, and wait for captureRate
		window.IMShow(*img)
		window.WaitKey(int(captureRate.Milliseconds()))
		return nil
	}
}

type ImageExporter struct {
	client pb.KafkaPixyClient
	topic  string
}

func NewImageExporter(endpoint string, tls bool, basicAuthUser, basicAuthPass, topic string) (*ImageExporter, error) {
	var opts []grpc.DialOption

	if tls {
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(nil)))
	} else {
		opts = append(opts, grpc.WithInsecure())
	}

	opts = append(opts, grpc.WithPerRPCCredentials(&GRPCClientBasicAuth{basicAuthUser, basicAuthPass}))

	conn, err := grpc.Dial(endpoint, opts...)
	if err != nil {
		return nil, fmt.Errorf("could not dial grpc: %w", err)
	}

	client := pb.NewKafkaPixyClient(conn)

	return &ImageExporter{
		client: client,
		topic:  topic,
	}, nil
}

func (e *ImageExporter) Send(img *gocv.Mat) error {
	resp, err := e.client.Produce(context.Background(), &pb.ProdRq{
		KeyUndefined: true,
		Topic:        e.topic,
		Message:      img.ToBytes(),
	})
	if err != nil {
		return fmt.Errorf("could not produce: %w", err)
	}
	log.Println("Meroxa response:", resp.String())
	return nil
}

type FaceRecognizer struct {
	webcam     *gocv.VideoCapture
	img        *gocv.Mat
	classifier *gocv.CascadeClassifier
}

func NewFaceRecognizer(deviceID int, classifierPath string) (*FaceRecognizer, error) {
	webcam, err := gocv.OpenVideoCapture(deviceID)
	if err != nil {
		return nil, fmt.Errorf("could not open video capture: %w", err)
	}

	// load classifier to recognize faces
	classifier := gocv.NewCascadeClassifier()
	if !classifier.Load(classifierPath) {
		_ = webcam.Close()
		return nil, fmt.Errorf("error reading classifier file: %w", classifierPath)
	}

	mat := gocv.NewMat()

	return &FaceRecognizer{
		webcam:     webcam,
		img:        &mat,
		classifier: &classifier,
	}, nil
}

func (fr *FaceRecognizer) Close() {
	err := fr.classifier.Close()
	if err != nil {
		log.Printf("could not close classifier: %s\n", err)
	}
	err = fr.img.Close()
	if err != nil {
		log.Printf("could not close image matrix: %s\n", err)
	}
	err = fr.webcam.Close()
	if err != nil {
		log.Printf("could not close video capture: %s\n", err)
	}
}

func (fr *FaceRecognizer) Detect(rate time.Duration) (*gocv.Mat, []image.Rectangle, error) {
	for {
		if ok := fr.webcam.Read(fr.img); !ok {
			return nil, nil, errors.New("cannot read device")
		}
		if fr.img.Empty() {
			time.Sleep(rate)
			continue
		}

		// detect faces
		rects := fr.classifier.DetectMultiScale(*fr.img)
		if len(rects) == 0 {
			// no faces detected
			time.Sleep(rate)
			continue
		}

		return fr.img, rects, nil
	}
}

type GRPCClientBasicAuth struct {
	Username string
	Password string
}

func (b GRPCClientBasicAuth) GetRequestMetadata(ctx context.Context, in ...string) (map[string]string, error) {
	auth := b.Username + ":" + b.Password
	enc := base64.StdEncoding.EncodeToString([]byte(auth))

	return map[string]string{
		"authorization": "Basic " + enc,
	}, nil
}

func (GRPCClientBasicAuth) RequireTransportSecurity() bool {
	return false
}
