package main

import (
	"context"
	"fmt"
	eciv1 "github.com/capitalonline/eci-manager/api/v1"
	"github.com/xuri/excelize/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"

	"k8s.io/client-go/dynamic"
	"log"
	"os"
)

func main() {
	ExportRecords()
}

func ExportRecords() {
	//kubeconfig := "/etc/kubernetes/admin.conf"
	config, err := rest.InClusterConfig()
	if err != nil {
		fmt.Printf("Error building kubeconfig: %s\n", err.Error())
		os.Exit(1)
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		log.Fatalf("Error creating clientset: %s", err.Error())
	}

	crdGV := schema.GroupVersionResource{
		Group:    "eci.eci.cds",
		Version:  "v1",
		Resource: "podrecords",
	}

	crs, err := dynamicClient.Resource(crdGV).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Fatalf("Error getting CRD: %s", err.Error())
	}
	var records = make([]eciv1.PodRecord, 0, 10)
	for _, cr := range crs.Items {
		record := eciv1.PodRecord{}
		if err = runtime.DefaultUnstructuredConverter.FromUnstructured(cr.Object, &record); err != nil {
			log.Fatalf("Error convert CRD: %s", err.Error())
		}
		fmt.Printf("record is %v", record)
		records = append(records, record)
	}
	if err = WriteToExcel(records); err != nil {
		log.Fatal(err)
	}
}

func WriteToExcel(records []eciv1.PodRecord) error {
	file := excelize.NewFile()
	defer file.Close()
	_, err := file.NewSheet("eci-podrecord")
	if err != nil {
		return err
	}
	row := []string{
		"UserID", "PodID", "PodName", "CpuRequest", "MemRequest", "CpuLimit", "MemLimit", "Gpu", "Node", "NodeMem", "NodeCpu", "StartTime", "EndTime", "EndStatus",
	}
	if err = file.SetSheetRow("eci-podrecord", "A1", &row); err != nil {
		return err
	}
	for i := 0; i < len(records); i++ {
		cell, err := excelize.CoordinatesToCellName(1, i+2)
		if err != nil {
			return err
		}
		record := records[i]
		row := []string{
			record.Spec.UserID,
			record.Spec.PodID,
			record.Spec.PodName,
			record.Spec.CpuRequest,
			record.Spec.MemRequest,
			record.Spec.CpuLimit,
			record.Spec.MemLimit,
			fmt.Sprintf("%d", record.Spec.Gpu),
			record.Spec.Node,
			record.Spec.NodeMem,
			record.Spec.NodeCpu,
			record.Spec.StartTime,
			record.Spec.EndTime,
			record.Spec.EndStatus,
		}
		if err = file.SetSheetRow("eci-podrecord", cell, &row); err != nil {
			return err
		}
	}
	return file.SaveAs("/root/eci-podrecord.xlsx")
}
