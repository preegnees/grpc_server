package errors

import "fmt"

var ErrInvalidIdChannelWhenRemove error = fmt.Errorf("Ошибка при удалении пира, такого канала не существует")
var ErrInvalidPeerWhenRemove error = fmt.Errorf("Ошибка при удалении пира, такого пира не существует")
var ErrInvalidAllowedNamesWhenSave error = fmt.Errorf("Ошибка при сохранении пира, allowedNames подключающегося клиентя не соответсвует allowedNames клиента в базе")

var ErrInvalidIdChannel error = fmt.Errorf("Invalid IdChannel, длинна должна быть > 8") 
var ErrInvalidIdName error = fmt.Errorf("Invalid Name, не должно быть пустым")
var ErrNoMetadata error = fmt.Errorf("$Нет метаданных")
var ErrGetMetadata error = fmt.Errorf("Error with metadata")