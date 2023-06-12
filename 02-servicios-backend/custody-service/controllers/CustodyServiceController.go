package controllers

import (
	"context"
	"errors"
	"regexp"
	"fmt"

	pb "github.com/malarcon-79/microservices-lab/grpc-protos-go/system/custody"
	"github.com/malarcon-79/microservices-lab/orm-go/dao"
	"github.com/malarcon-79/microservices-lab/orm-go/model"
	"github.com/jinzhu/gorm"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Controlador de servicio gRPC
type CustodyServiceController struct {
	logger *zap.SugaredLogger // Logger
	re     *regexp.Regexp     // Expresión regular para validar formato de períodos YYYY-MM
}

// Método a nivel de package, permite construir una instancia correcta del controlador de servicio gRPC
func NewCustodyServiceController() (CustodyServiceController, error) {
	_logger, _ := zap.NewProduction() // Generamos instancia de logger
	logger := _logger.Sugar()

	re, err := regexp.Compile(`^\d{4}\-(0?[1-9]|1[012])$`) // Expresión regular para validar períodos YYYY-MM
	if err != nil {
		return CustodyServiceController{}, err
	}

	instance := CustodyServiceController{
		logger: logger, // Asignamos el logger
		re:     re,     // Asignamos el RegExp precompilado
	}
	return instance, nil // Devolvemos la nueva instancia de este Struct y un puntero nulo para el error
}

func (c *CustodyServiceController) AddCustodyStock(ctx context.Context, msg *pb.CustodyAdd) (*pb.Empty, error) {
	// Implementar este método
	// Verificar que los campos obligatorios no sean nulos o inválidos
	// Verificar que los campos obligatorios no sean nulos o inválidos
	if msg.Period == "" {
		return nil, status.Errorf(codes.InvalidArgument, "El campo 'period' es requerido")
	}
	if msg.Stock == "" {
		return nil, status.Errorf(codes.InvalidArgument, "El campo 'stock' es requerido")
	}
	if msg.ClientId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "El campo 'client_id' es requerido")
	}
	if msg.Quantity <= 0 {
		return nil, status.Errorf(codes.InvalidArgument, "El campo 'quantity' debe ser mayor o igual a cero")
	}

	// Validar formato de período YYYY-MM
	validated_period, err := regexp.MatchString(`^\d{4}-\d{2}$`, msg.Period)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Error al validar el formato de período")
	}
	if !validated_period {
		return nil, status.Errorf(codes.InvalidArgument, "Formato de período inválido")
	}

    // Buscar la custodia existente en la base de datos
	existingCustody := &model.Custody{}
	orm := dao.DB.Model(&model.Custody{})
	if err := orm.Where(&model.Custody{
		Period:   msg.Period,
		Stock:    msg.Stock,
		ClientId: msg.ClientId,
	}).First(existingCustody).Error; err != nil {
		
		if !gorm.IsRecordNotFoundError(err) {
			// Error al buscar la custodia
			c.logger.Errorf("no se pudo buscar la custodia existente: %v", err)
			return nil, status.Errorf(codes.Internal, "no se pudo realizar la búsqueda de la custodia")
		}

    // No se encontró la custodia existente, crear una nueva
    newCustody := &model.Custody{
        Period:   msg.Period,
        Stock:    msg.Stock,
        ClientId: msg.ClientId,
        Quantity: int32(msg.Quantity),
    }

    // Guardar la nueva custodia en la base de datos
    if err := orm.Create(newCustody).Error; err != nil {
        c.logger.Errorf("no se pudo crear la nueva custodia: %v", err)
        return nil, status.Errorf(codes.Internal, "no se pudo crear la custodia")
    }
} else {
    // Se encontró la custodia existente, actualizar la cantidad
    existingCustody.Quantity += int32(msg.Quantity)

    // Actualizar la custodia en la base de datos
    if err := orm.Save(existingCustody).Error; err != nil {
        c.logger.Errorf("no se pudo actualizar la custodia existente: %v", err)
        return nil, status.Errorf(codes.Internal, "no se pudo actualizar la custodia")
    }
}

// Retornar una respuesta vacía
return &pb.Empty{}, nil

}

func (c *CustodyServiceController) ClosePeriod(ctx context.Context, msg *pb.CloseFilters) (*pb.Empty, error) {
	return nil, errors.New("no implementado")
}

// Método que obtiene la custodia mediante criterio de búsqueda
func (c *CustodyServiceController) GetCustody(ctx context.Context, msg *pb.CustodyFilter) (*pb.Custodies, error) {
	// Implementar este método
	// Con esta línea instanciamos el ORM para trabajar con la tabla "Custody"
	orm := dao.DB.Model(&model.Custody{})

	// Arreglo de punteros a registros de tabla "Custody"
	custodies := []*model.Custody{}
	
	// Creamos el filtro de búsqueda usando los campos del mismo modelo
	filter := &model.Custody{
		Period:        msg.Period,
		Stock:         msg.Stock,
		ClientId:      msg.ClientId,
	}
	// Ejecutamos el SELECT y evaluamos si hubo errores
	if err := orm.Find(&custodies, filter).Error; err != nil {
		c.logger.Errorf("no se pudo buscar custodias con filtros %v", filter, err)
		return nil, status.Errorf(codes.Internal, "no se pudo realizar query")
	}

	// Este será el mensaje de salida
	result := &pb.Custodies{}
	// Iteramos el arreglo de registros del SELECT anterior.
	// En Go, la instrucción "for range" nos permite recorrer estructuras iterables de forma simple.
	// El primer elemento es el índice, y el segundo es el ítem iterado.
	// La instrucción "_" (guión bajo) indica que se puede ignorar la asignación de ese valor
	// Iteramos sobre los registros del SELECT anterior

	for _, custody := range custodies {
		
		custodyItems := []*pb.Custodies_Custody{
			&pb.Custodies_Custody{
				Period:   custody.Period,
				Stock:    custody.Stock,
				ClientId: custody.ClientId,
				Market:   custody.Market,
				Price:    custody.Price.InexactFloat64(),
				Quantity: custody.Quantity,
			},
		}
		
		result.Items = append(result.Items, custodyItems...)
		fmt.Println(result.Items)
	}
	// Retornamos la respuesta correcta
	
	return result, nil
}
