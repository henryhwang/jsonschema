package main

import (
    "encoding/json"
    "io/ioutil"
    "log"
    "os"
    "path/filepath"

    "google.golang.org/protobuf/proto"
    "google.golang.org/protobuf/reflect/protoreflect"
    "google.golang.org/protobuf/types/descriptorpb"
)

func main() {
    // Directory containing .proto files
    rootDir := "./" // Change this to your project's root directory

    schema := map[string]interface{}{
        "type":       "object",
        "properties": make(map[string]interface{}),
    }

    err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }
        if !info.IsDir() && filepath.Ext(path) == ".proto" {
            if err := collectSchemaFromProto(path, schema); err != nil {
                return err
            }
        }
        return nil
    })

    if err != nil {
        log.Fatalf("Failed to walk through directory: %v", err)
    }

    jsonSchema, err := json.MarshalIndent(schema, "", "  ")
    if err != nil {
        log.Fatalf("Failed to marshal schema to JSON: %v", err)
    }

    err = ioutil.WriteFile("config_schema.json", jsonSchema, 0644)
    if err != nil {
        log.Fatalf("Failed to write JSON schema file: %v", err)
    }

    log.Println("JSON schema has been generated as 'config_schema.json'")
}

func collectSchemaFromProto(protoFilePath string, schema map[string]interface{}) error {
    fileDescriptorSet, err := proto.LoadDescriptorSetFromFile(protoFilePath)
    if err != nil {
        return err
    }

    for _, file := range fileDescriptorSet.GetFile() {
        for _, message := range file.GetMessageTypes() {
            processMessageForSchema(message, schema)
        }
    }

    return nil
}

func processMessageForSchema(md protoreflect.MessageDescriptor, schema map[string]interface{}) {
    properties := schema["properties"].(map[string]interface{})
    md.Fields().Range(func(fd protoreflect.FieldDescriptor) bool {
        fieldSchema := map[string]interface{}{
            "type": mapProtoTypeToJSONType(fd.Kind()),
        }

        if fd.IsList() {
            fieldSchema["type"] = "array"
            fieldSchema["items"] = map[string]interface{}{"type": mapProtoTypeToJSONType(fd.Kind())}
        }

        if fd.IsMap() {
            fieldSchema["type"] = "object"
            fieldSchema["additionalProperties"] = map[string]interface{}{
                "type": mapProtoTypeToJSONType(fd.MapValue().Kind()),
            }
        }

        properties[string(fd.Name())] = fieldSchema
        return true
    })
}

func mapProtoTypeToJSONType(kind protoreflect.Kind) string {
    switch kind {
    case protoreflect.StringKind:
        return "string"
    case protoreflect.Int32Kind, protoreflect.Int64Kind, protoreflect.Sint32Kind, protoreflect.Sint64Kind:
        return "integer"
    case protoreflect.FloatKind, protoreflect.DoubleKind:
        return "number"
    case protoreflect.BoolKind:
        return "boolean"
    case protoreflect.BytesKind:
        return "string" // JSON has no native byte array, so string might be the closest
    case protoreflect.MessageKind, protoreflect.GroupKind:
        return "object"
    case protoreflect.EnumKind:
        return "string" // Treating enum as string in JSON for simplicity
    default:
        return "object" // Catch-all for unhandled types, should be rare
    }
}